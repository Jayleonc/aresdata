package utils

import "time"

// TimeToPtr returns a pointer to a time.Time value.
func TimeToPtr(t time.Time) *time.Time {
	return &t
}
