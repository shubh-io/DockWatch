package tui

import (
	"strconv"
	"strings"
)

func parsePercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseNetIO(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "─" {
		return 0
	}
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return 0
	}
	total := 0.0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v := parseSize(p)
		total += v
	}
	return total
}

func parseSize(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// remove possible commas
	s = strings.ReplaceAll(s, ",", "")
	// split number and unit
	num := ""
	unit := ""
	for i, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			num += string(r)
		} else {
			unit = strings.TrimSpace(s[i:])
			break
		}
	}
	if num == "" {
		return 0
	}
	val, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	unit = strings.ToLower(strings.TrimSpace(unit))
	switch unit {
	case "b", "bytes", "byte", "":
		return val
	case "kb", "kib":
		return val * 1000
	case "mb", "mib":
		return val * 1000 * 1000
	case "gb", "gib":
		return val * 1000 * 1000 * 1000
	default:
		if strings.HasSuffix(unit, "b") {
			prefix := strings.TrimSuffix(unit, "b")
			if prefix == "k" {
				return val * 1000
			}
			if prefix == "m" {
				return val * 1000 * 1000
			}
			if prefix == "g" {
				return val * 1000 * 1000 * 1000
			}
		}
	}
	return val
}
