package libcore

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestSha(t *testing.T) {
	// known vectors for "abc"
	if got := hex.EncodeToString(Sha1([]byte("abc"))); got != "a9993e364706816aba3e25717850c26c9cd0d89d" {
		t.Fatalf("Sha1: %s", got)
	}
	if got := Sha256Hex([]byte("abc")); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("Sha256Hex: %s", got)
	}
	want1 := sha1.Sum(nil)
	if !bytes.Equal(Sha1(nil), want1[:]) {
		t.Fatal("Sha1 of empty input")
	}
	want256 := sha256.Sum256(nil)
	if Sha256Hex(nil) != hex.EncodeToString(want256[:]) {
		t.Fatal("Sha256Hex of empty input")
	}
}

func TestWrapString(t *testing.T) {
	if wrapString("x").Value != "x" || wrapString("").Value != "" {
		t.Fatal("wrapString lost value")
	}
}

func TestUnxz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.xz")
	var buf bytes.Buffer
	w, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("hello xz"))
	w.Close()
	os.WriteFile(archive, buf.Bytes(), 0o644)

	out := filepath.Join(dir, "out")
	if err := Unxz(archive, out); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); string(got) != "hello xz" {
		t.Fatalf("got %q", got)
	}

	if Unxz(filepath.Join(dir, "missing.xz"), out) == nil {
		t.Fatal("missing archive must fail")
	}
	bad := filepath.Join(dir, "bad.xz")
	os.WriteFile(bad, []byte("not xz data at all"), 0o644)
	if Unxz(bad, filepath.Join(dir, "bad.out")) == nil {
		t.Fatal("corrupt archive must fail")
	}
}

func TestUnzip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Create("sub/")
	f, _ := zw.Create("sub/file.txt")
	f.Write([]byte("zipped"))
	f, _ = zw.Create("root.txt")
	f.Write([]byte("root"))
	zw.Close()
	os.WriteFile(archive, buf.Bytes(), 0o644)

	out := filepath.Join(dir, "out")
	if err := Unzip(archive, out); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"sub/file.txt": "zipped", "root.txt": "root"} {
		if got, err := os.ReadFile(filepath.Join(out, name)); err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v", name, got, err)
		}
	}
	if Unzip(filepath.Join(dir, "missing.zip"), out) == nil {
		t.Fatal("missing archive must fail")
	}
}

func TestHttpClientRequest(t *testing.T) {
	var gotMethod, gotUA, gotHeader, gotBody, gotUser string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			http.Error(w, strings.Repeat("e", 200), http.StatusTeapot)
			return
		}
		gotMethod, gotUA, gotHeader = r.Method, r.Header.Get("User-Agent"), r.Header.Get("X-Test")
		gotUser, _, _ = r.BasicAuth()
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("X-Reply", "yes")
		w.Write([]byte("response body"))
	}))
	defer server.Close()

	client := NewHttpClient()
	client.ModernTLS()
	defer client.Close()

	req := client.NewRequest()
	if err := req.SetURL(strings.Replace(server.URL, "http://", "http://bob:pw@", 1) + "/ok"); err != nil {
		t.Fatal(err)
	}
	req.SetMethod("POST")
	req.SetUserAgent("ua/1")
	req.SetHeader("X-Test", "v")
	req.SetContentString("payload")
	resp, err := req.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "POST" || gotUA != "ua/1" || gotHeader != "v" || gotBody != "payload" || gotUser != "bob" {
		t.Fatalf("server saw %q %q %q %q %q", gotMethod, gotUA, gotHeader, gotBody, gotUser)
	}
	if resp.GetHeader("X-Reply").Value != "yes" {
		t.Fatal("missing response header")
	}
	s, err := resp.GetContentString()
	if err != nil || s.Value != "response body" {
		t.Fatalf("%v %v", s, err)
	}
	// content is cached across calls
	if c, err := resp.GetContent(); err != nil || string(c) != "response body" {
		t.Fatalf("second read: %q %v", c, err)
	}

	// non-200 becomes an error with a truncated body
	req = client.NewRequest()
	req.SetURL(server.URL + "/fail")
	_, err = req.Execute()
	if err == nil || !strings.HasPrefix(err.Error(), "HTTP 418") || !strings.HasSuffix(err.Error(), " ...") {
		t.Fatalf("unexpected error %v", err)
	}

	if req.SetURL("http://[::1") == nil {
		t.Fatal("bad URL must fail")
	}
}

func TestHttpResponseWriteTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("file content"))
	}))
	defer server.Close()
	req := NewHttpClient().NewRequest()
	req.SetURL(server.URL)
	resp, err := req.Execute()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dl")
	if err = resp.WriteTo(path); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "file content" {
		t.Fatalf("got %q", got)
	}
}

func TestHttpPinnedSHA256(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	sum := sha256.Sum256(server.Certificate().Raw)

	for _, tc := range []struct {
		pin string
		ok  bool
	}{{hex.EncodeToString(sum[:]), true}, {strings.Repeat("0", 64), false}} {
		client := NewHttpClient()
		client.PinnedSHA256(tc.pin)
		req := client.NewRequest()
		req.AllowInsecure() // chain is self-signed; only the pin is under test
		req.SetURL(server.URL)
		_, err := req.Execute()
		if (err == nil) != tc.ok {
			t.Fatalf("pin %s: err=%v want ok=%v", tc.pin[:4], err, tc.ok)
		}
	}
}
