package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	pathSeparatorArrow = " -> "
	pathSeparatorPipe  = " | "
	pathEllipsis       = "..."
)

// collapsePathForDisplay shortens hierarchy paths while preserving the first and focused final segment.
func collapsePathForDisplay(path string, max int) string {
	path = strings.TrimSpace(path)
	if max <= 0 || path == "" {
		return ""
	}
	separator := detectPathSeparator(path)
	if separator == "" {
		return truncate(path, max)
	}
	segments := splitPathSegments(path, separator)
	if len(segments) == 0 {
		return truncate(path, max)
	}
	if len(segments) == 1 {
		return truncate(segments[0], max)
	}
	if len(segments) == 2 {
		return collapseTwoSegmentPath(segments[0], segments[1], separator, max)
	}
	return collapseMultiSegmentPath(segments, separator, max)
}

// detectPathSeparator resolves the hierarchy separator used in the provided path string.
func detectPathSeparator(path string) string {
	if strings.Contains(path, pathSeparatorArrow) {
		return pathSeparatorArrow
	}
	if strings.Contains(path, pathSeparatorPipe) {
		return pathSeparatorPipe
	}
	return ""
}

// splitPathSegments trims and normalizes split hierarchy path segments.
func splitPathSegments(path, separator string) []string {
	rawSegments := strings.Split(path, separator)
	segments := make([]string, 0, len(rawSegments))
	for _, raw := range rawSegments {
		segment := strings.TrimSpace(raw)
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}

// collapseTwoSegmentPath clamps a two-segment path by trimming the leading segment first.
func collapseTwoSegmentPath(first, last, separator string, max int) string {
	full := first + separator + last
	if pathDisplayWidth(full) <= max {
		return full
	}
	suffix := separator + last
	firstBudget := max - pathDisplayWidth(suffix)
	if firstBudget > 0 {
		return truncate(first, firstBudget) + suffix
	}
	return truncate(last, max)
}

// collapseMultiSegmentPath removes middle hierarchy segments from left to right before endpoint clamping.
func collapseMultiSegmentPath(segments []string, separator string, max int) string {
	full := strings.Join(segments, separator)
	if pathDisplayWidth(full) <= max {
		return full
	}

	first := segments[0]
	last := segments[len(segments)-1]
	middle := append([]string(nil), segments[1:len(segments)-1]...)
	for removed := 1; removed <= len(middle); removed++ {
		candidateSegments := []string{first, pathEllipsis}
		candidateSegments = append(candidateSegments, middle[removed:]...)
		candidateSegments = append(candidateSegments, last)
		candidate := strings.Join(candidateSegments, separator)
		if pathDisplayWidth(candidate) <= max {
			return candidate
		}
	}

	compact := first + separator + pathEllipsis + separator + last
	if pathDisplayWidth(compact) <= max {
		return compact
	}

	suffix := separator + pathEllipsis + separator + last
	firstBudget := max - pathDisplayWidth(suffix)
	if firstBudget > 0 {
		return truncate(first, firstBudget) + suffix
	}

	prefix := truncate(first, 1) + separator + pathEllipsis + separator
	lastBudget := max - pathDisplayWidth(prefix)
	if lastBudget > 0 {
		return prefix + truncate(last, lastBudget)
	}

	fallback := pathEllipsis + separator + last
	if pathDisplayWidth(fallback) <= max {
		return fallback
	}
	return truncate(last, max)
}

// pathDisplayWidth returns the visual width used by path clamping logic.
func pathDisplayWidth(text string) int {
	return lipgloss.Width(text)
}
