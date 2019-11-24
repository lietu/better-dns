package stats

import "fmt"

func RequestPct(requests uint64, total uint64) string {
	var pct = 0.0
	if total > 0 {
		pct = float64(requests) / float64(total)
	}
	return fmt.Sprintf("%.1f%%", 100*pct)
}
