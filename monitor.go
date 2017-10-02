package microcache

import (
	"time"
)

// Monitor is an interface for collecting metrics about the microcache
type Monitor interface {
	GetInterval() time.Duration
	Log(MonitorStats)
	Start()
	Hit()
	Miss()
	Error()
}

type MonitorStats struct {
	Size int
	HitRate float64
	ErrorRate float64
}
