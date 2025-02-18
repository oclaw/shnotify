package common

import "time"

type Clock interface {
	NowUnix() int64
}

type DefaultClock struct{}

func (*DefaultClock) NowUnix() int64 {
	now := time.Now()
	return now.Unix()
}
