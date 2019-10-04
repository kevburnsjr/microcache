package microcache

import (
	"context"
	"net/http"
)

// newBackgroundRequest clones a request for use in background object revalidation.
// This prevents a closed foreground request context from prematurely cancelling
// the background request context.
func newBackgroundRequest(r *http.Request) *http.Request {
	return r.Clone(bgContext{r.Context(), make(chan struct{})})
}

type bgContext struct {
	context.Context
	done chan struct{}
}

func (c bgContext) Done() <-chan struct{} {
	return c.done
}
