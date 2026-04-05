package cache

import "time"

func timeFromUnix(unix int64) time.Time {
	return time.Unix(unix, 0)
}

func newTimeFromUnix(unix int64) *time.Time {
	t := time.Unix(unix, 0)
	return &t
}
