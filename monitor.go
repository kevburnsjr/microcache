package microcache

import (
	"time"
)

type Monitor interface {
	GetInterval() time.Duration
	Log(objectCount int, size int, hitRate float64, missRate float64, errorRate float64)
	Start()
	Hit()
	Miss()
	Error()
}
