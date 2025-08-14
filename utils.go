package main

import "strings"


func extractKey(result string) string {
	if idx := strings.Index(result, "#"); idx != -1 {
		return result[:idx]
	}
	return result
}