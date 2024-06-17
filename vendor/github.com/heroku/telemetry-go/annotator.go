package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
)

// annotator is used to allow instrumented HTTP handlers to add custom attributes to
// the metrics recorded by the net/http instrumentation.
type annotator struct {
	mu         sync.Mutex
	name       string
	attributes []attribute.KeyValue
}

// addAttributes attributes to a Annotator.
func (a *annotator) addAttributes(ls ...attribute.KeyValue) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.attributes = append(a.attributes, ls...)
}

// getAttributes returns a copy of the attributes added to the Annotator.
func (a *annotator) getAttributes() []attribute.KeyValue {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]attribute.KeyValue, len(a.attributes))
	copy(out, a.attributes)

	return out
}

func (a *annotator) setName(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.name = name
}

func (a *annotator) getName() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.name
}

type annotatorContextKeyType int

const annotatorContextKey annotatorContextKeyType = 0

func injectAnnotator(ctx context.Context, a *annotator) context.Context {
	return context.WithValue(ctx, annotatorContextKey, a)
}

// annotatorFromContext retrieves a Annotator instance from the provided context if
// one is available.  If no Annotator was found in the provided context a new, empty
// Annotator is returned and the second return value is false.  In this case it is
// safe to use the Annotator but any attributes added to it will not be used.
func annotatorFromContext(ctx context.Context) (*annotator, bool) {
	a, ok := ctx.Value(annotatorContextKey).(*annotator)
	if !ok {
		a = &annotator{}
	}
	return a, ok
}
