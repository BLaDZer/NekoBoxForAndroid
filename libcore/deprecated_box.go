package libcore

import (
	"log"
	"sync"

	"github.com/sagernet/sing-box/experimental/deprecated"
)

// deprecatedReporter receives sing-box "deprecated feature" notes instead of
// letting them be dropped silently (no deprecated.Manager in the context).
// It never aborts: the stderr manager used by the stock CLI would Fatal() on
// notes that are about to be removed, which must not kill the Android service.
type deprecatedReporter struct {
	access   sync.Mutex
	reported map[string]deprecated.Note
}

func newDeprecatedReporter() *deprecatedReporter {
	return &deprecatedReporter{reported: make(map[string]deprecated.Note)}
}

func (r *deprecatedReporter) ReportDeprecated(feature deprecated.Note) {
	r.access.Lock()
	_, seen := r.reported[feature.Name]
	r.reported[feature.Name] = feature
	r.access.Unlock()
	if !seen {
		log.Println("sing-box deprecated:", feature.MessageWithLink())
	}
}

// Names returns the names of all deprecated features reported so far.
func (r *deprecatedReporter) Names() []string {
	r.access.Lock()
	defer r.access.Unlock()
	names := make([]string, 0, len(r.reported))
	for name := range r.reported {
		names = append(names, name)
	}
	return names
}
