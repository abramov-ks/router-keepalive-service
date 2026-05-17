package model

import "time"

type PingEvent struct {
	ID         int64
	RouterID   string
	ReceivedAt time.Time
	SourceIP   string
}

type RouterInfo struct {
	RouterID   string    `json:"router_id"`
	LastSeen   time.Time `json:"last_seen"`
	FirstSeen  time.Time `json:"first_seen"`
	TotalPings int64     `json:"total_pings"`
}

type BucketPoint struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

type DayPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
