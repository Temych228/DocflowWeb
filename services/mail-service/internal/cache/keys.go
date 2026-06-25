package cache

import (
	"fmt"
	"strings"
)

func JobKey(jobID string) string {
	return "mail:job:" + strings.TrimSpace(jobID)
}

func DedupKey(jobID string) string {
	return "mail:dedup:" + strings.TrimSpace(jobID)
}

func StatsHourKey(hour string) string {
	return "mail:stats:hour:" + strings.TrimSpace(hour)
}

func StatsCategoryHourKey(category, hour string) string {
	return fmt.Sprintf("mail:stats:category:%s:%s", strings.TrimSpace(category), strings.TrimSpace(hour))
}

func StatsStatusHourKey(status, hour string) string {
	return fmt.Sprintf("mail:stats:status:%s:%s", strings.TrimSpace(status), strings.TrimSpace(hour))
}
