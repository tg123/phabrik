package common

import (
	"math"
	"time"
)

type StopwatchTime int64

type DateTime int64

type TimeSpan int64

const (
	TimeSpanMax TimeSpan = math.MaxInt64
)

func TimeSpanFromDuration(d time.Duration) TimeSpan {
	return TimeSpan(d.Nanoseconds() / 100)
}

func (s TimeSpan) ToDuration() time.Duration {
	return time.Duration(s) * 100
}
