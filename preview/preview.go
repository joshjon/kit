package preview

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultMaxChars   = 35
	DefaultMaxInspect = 4096 // cap work per message
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// Preview generates a human-friendly preview string for arbitrary message bytes.
// It is designed for high-volume usage:
//   - Caps work to maxInspect bytes (O(min(n, maxInspect)))
//   - Avoids []rune allocations for truncation
//   - Only compacts JSON that starts with '{' or '['
//   - Uses a buffer pool for JSON compaction
func Preview(b []byte, maxChars int, maxInspect int) string {
	if maxChars <= 0 || len(b) == 0 {
		return ""
	}
	if maxInspect > 0 && len(b) > maxInspect {
		b = b[:maxInspect]
	}

	// If it's not valid UTF-8, treat as binary
	if !utf8.Valid(b) {
		return binaryPreview(b, maxChars)
	}

	s := string(b)

	// If it has lots of control/non-graphic chars, treat as binary
	if !looksMostlyPrintable(s) {
		return binaryPreview(b, maxChars)
	}

	// Trim (cheap) and early exit
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// If it looks like JSON try compacting so multiline JSON becomes one line.
	// Only attempt if it starts with '{' or '[' to reduce pointless work.
	if looksLikeJSONStart(s) {
		if compacted, ok := tryCompactJSON(s); ok {
			s = compacted
		}
	}

	// Collapse whitespace/newlines/tabs into single spaces
	s = collapseWhitespace(s)

	return truncateRunesNoAlloc(s, maxChars)
}

func looksLikeJSONStart(s string) bool {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func tryCompactJSON(s string) (string, bool) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := json.Compact(buf, []byte(s)); err != nil {
		return "", false
	}
	return buf.String(), true
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inWS := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWS = true
			continue
		}
		if inWS && b.Len() > 0 {
			b.WriteByte(' ')
		}
		inWS = false
		b.WriteRune(r)
	}
	return b.String()
}

func truncateRunesNoAlloc(s string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	if maxChars == 1 {
		// If there's any content, show ellipsis
		if strings.TrimSpace(s) == "" {
			return ""
		}
		return "…"
	}

	// Count runes up to maxChars. Stop early without allocating []rune.
	// If we exceed, we return first (maxChars-1) runes + ellipsis.
	runeCount := 0
	cutByteIdx := -1

	for i := range s {
		if runeCount == maxChars-1 {
			cutByteIdx = i
			break
		}
		runeCount++
	}

	// If we never hit the cut point, the string is <= maxChars-1 runes.
	// But we still need to know if it's longer than maxChars runes total.
	if cutByteIdx == -1 {
		// Count total runes up to maxChars to see if it fits
		total := 0
		for range s {
			total++
			if total > maxChars {
				break
			}
		}
		if total <= maxChars {
			return s
		}
		// Need to cut at maxChars-1: find it again (still tiny maxChars)
		total = 0
		for i := range s {
			if total == maxChars-1 {
				return s[:i] + "…"
			}
			total++
		}
		return "…"
	}

	// Now determine if the string continues beyond cutByteIdx (i.e. more runes exist)
	rest := s[cutByteIdx:]
	if rest == "" {
		return s // exactly maxChars-1 runes, no need for ellipsis
	}
	// If there's at least one rune in rest, we exceeded maxChars-1 runes.
	// But if total runes <= maxChars, we should return original.
	total := 0
	for range s {
		total++
		if total > maxChars {
			break
		}
	}
	if total <= maxChars {
		return s
	}

	return s[:cutByteIdx] + "…"
}

func looksMostlyPrintable(s string) bool {
	// Treat as printable if >= 85% of non-space runes are graphic.
	var total, printable int
	for _, r := range s {
		if r == '\uFFFD' { // replacement char often indicates decoding issues
			total++
			continue
		}
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if unicode.IsGraphic(r) {
			printable++
		}
	}
	if total == 0 {
		return true
	}
	return float64(printable)/float64(total) >= 0.85
}

func binaryPreview(b []byte, maxChars int) string {
	// Example: "<binary 123B> 0a1b2c3d…"
	head := 12
	if len(b) < head {
		head = len(b)
	}
	hexHead := hex.EncodeToString(b[:head])
	msg := fmt.Sprintf("<binary %dB> %s", len(b), hexHead)
	return truncateRunesNoAlloc(msg, maxChars)
}
