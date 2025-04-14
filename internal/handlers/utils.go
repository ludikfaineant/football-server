package handlers

import (
	"fmt"
	"strconv"
)

func formatSeason(season string) string {
	if season == "" {
		return ""
	}
	startYear, _ := strconv.Atoi(season)
	endYear := startYear + 1
	return fmt.Sprintf("%d/%02d", startYear, endYear%100) // 2023 → 2023/24
}
