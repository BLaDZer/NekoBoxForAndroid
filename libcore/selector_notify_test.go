package libcore

import (
	"github.com/matsuridayo/libneko/neko_log"
	"sync"
	"testing"
	"time"
)

// selectorTestPlatform adds what a started sing-box calls on the platform.
type selectorTestPlatform struct{ monitorTestPlatform }

func (p *selectorTestPlatform) WIFIState() string { return "" }

type selectionRecorder struct {
	NB4AInterface
	access sync.Mutex
	events [][2]string
	notify chan struct{}
}

func (r *selectionRecorder) Selector_OnProxySelected(selectorTag, tag string) {
	r.access.Lock()
	r.events = append(r.events, [2]string{selectorTag, tag})
	r.access.Unlock()
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

func (r *selectionRecorder) snapshot() [][2]string {
	r.access.Lock()
	defer r.access.Unlock()
	return append([][2]string(nil), r.events...)
}

func (r *selectionRecorder) wait(t *testing.T, want int) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for len(r.snapshot()) < want {
		select {
		case <-r.notify:
		case <-deadline:
			t.Fatalf("timed out waiting for %d selection events, got %v", want, r.snapshot())
		}
	}
}

const selectorTestConfig = `{
  "outbounds": [
    {"type": "selector", "tag": "proxy", "outbounds": ["a", "b", "c"], "default": "a"},
    {"type": "direct", "tag": "a"},
    {"type": "direct", "tag": "b"},
    {"type": "direct", "tag": "c"}
  ],
  "route": {"final": "proxy"}
}`

// Stock sing-box has no selection hook, so libcore reports selection changes
// itself: directly after BoxInstance.SelectOutbound and by polling, which also
// catches changes made behind our back (e.g. Clash API -> Selector.SelectOutbound).
func TestSelectorNotification(t *testing.T) {
	previousBox, previousApp, previousInterval := intfBox, intfNB4A, selectorPollInterval
	recorder := &selectionRecorder{notify: make(chan struct{}, 8)}
	intfBox, intfNB4A, selectorPollInterval = &selectorTestPlatform{}, recorder, 20*time.Millisecond
	// neko_log.LogWriter is only set up by InitCore; with logging disabled its Write is a no-op.
	previousLogDisable := neko_log.LogWriterDisable
	neko_log.LogWriterDisable = true
	t.Cleanup(func() {
		intfBox, intfNB4A, selectorPollInterval = previousBox, previousApp, previousInterval
		neko_log.LogWriterDisable = previousLogDisable
	})

	instance, err := NewSingBoxInstance(selectorTestConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	if err = instance.Start(); err != nil {
		t.Fatal(err)
	}

	// initial selection is recorded, not reported
	time.Sleep(100 * time.Millisecond)
	if events := recorder.snapshot(); len(events) != 0 {
		t.Fatalf("initial selection must not be reported, got %v", events)
	}

	// selection through libcore
	if !instance.SelectOutbound("b") {
		t.Fatal("SelectOutbound(b) failed")
	}
	recorder.wait(t, 1)
	if got := recorder.snapshot()[0]; got != [2]string{"proxy", "b"} {
		t.Fatalf("unexpected event %v", got)
	}

	// re-selecting the same outbound and unknown tags report nothing
	if !instance.SelectOutbound("b") || instance.SelectOutbound("missing") {
		t.Fatal("unexpected SelectOutbound result")
	}

	// selection behind libcore's back (what the Clash API does)
	if !instance.selector.SelectOutbound("c") {
		t.Fatal("selector.SelectOutbound(c) failed")
	}
	recorder.wait(t, 2)

	time.Sleep(100 * time.Millisecond)
	events := recorder.snapshot()
	if len(events) != 2 || events[1] != [2]string{"proxy", "c"} {
		t.Fatalf("want exactly [proxy b] then [proxy c], got %v", events)
	}
}

// Close must wait for watchSelector, so no callback reaches the app afterwards.
func TestSelectorNoNotificationAfterClose(t *testing.T) {
	previousBox, previousApp, previousInterval := intfBox, intfNB4A, selectorPollInterval
	recorder := &selectionRecorder{notify: make(chan struct{}, 8)}
	intfBox, intfNB4A, selectorPollInterval = &selectorTestPlatform{}, recorder, 5*time.Millisecond
	previousLogDisable := neko_log.LogWriterDisable
	neko_log.LogWriterDisable = true
	t.Cleanup(func() {
		intfBox, intfNB4A, selectorPollInterval = previousBox, previousApp, previousInterval
		neko_log.LogWriterDisable = previousLogDisable
	})

	instance, err := NewSingBoxInstance(selectorTestConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = instance.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	selector := instance.selector
	exited := instance.selectorExited
	// change selection behind libcore's back, then close immediately
	selector.SelectOutbound("c")
	if err = instance.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	default:
		t.Fatal("watchSelector still running after Close")
	}
	before := len(recorder.snapshot())
	time.Sleep(100 * time.Millisecond)
	if after := len(recorder.snapshot()); after != before {
		t.Fatalf("callbacks after Close: %d -> %d", before, after)
	}
}

func TestNotifySelectedWithoutSelector(t *testing.T) {
	previousApp := intfNB4A
	recorder := &selectionRecorder{notify: make(chan struct{}, 1)}
	intfNB4A = recorder
	t.Cleanup(func() { intfNB4A = previousApp })
	(&BoxInstance{}).notifySelected() // must not panic or report
	if len(recorder.snapshot()) != 0 {
		t.Fatal("unexpected event")
	}
}
