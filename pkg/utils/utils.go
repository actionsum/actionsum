package utils

import "fmt"

func FormatRoundedUnit(seconds int64) string {
	if seconds < 0 {
		seconds = -seconds
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds > 3600 {
		return fmt.Sprintf("%dh", int64(seconds/3600))
	}
	return fmt.Sprintf("%dm", int64(seconds/60))
}
