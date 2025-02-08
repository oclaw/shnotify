package main

import "time"

type Clock interface {
	NowUnix() int64
}

type defaultClock struct{}

func (*defaultClock) NowUnix() int64 {
	now := time.Now()
	return now.Unix()
}
