package microcache

import (
	"time"
)

// Monitor is an interface for collecting metrics about the microcache
type Monitor interface {
	GetInterval() time.Duration
	Log(Stats)
	Hit()
	Miss()
	Error()
}

type Stats struct {
	Size   int
	Hits   int
	Misses int
	Stales int
	Errors int
}
