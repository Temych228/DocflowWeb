package cache

import (
	"fmt"
	"strings"
)

func UnreadKey(userID string) string {
	return "notif:unread:" + strings.TrimSpace(userID)
}

func PrefsKey(userID string) string {
	return "notif:prefs:" + strings.TrimSpace(userID)
}

func DedupKey(eventKey string) string {
	return "notif:dedup:" + strings.TrimSpace(eventKey)
}

func HourStatsKey(hour string) string {
	return "notif:stats:hour:" + strings.TrimSpace(hour)
}

func DayStatsKey(day string) string {
	return "notif:stats:day:" + strings.TrimSpace(day)
}

func CategoryDayStatsKey(category, day string) string {
	return fmt.Sprintf("notif:stats:category:%s:%s", strings.TrimSpace(category), strings.TrimSpace(day))
}
