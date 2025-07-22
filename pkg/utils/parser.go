package utils

import "time"

// TimeToPtr returns a pointer to a time.Time value.
func TimeToPtr(t time.Time) *time.Time {
	return &t
}

// ParseTimeRFC3339 parses an RFC3339 string to time.Time. If parsing fails, returns zero value.
func ParseTimeRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
