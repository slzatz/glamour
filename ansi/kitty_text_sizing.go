package ansi

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// KittyTextSizingSupport tracks whether the terminal supports Kitty's OSC 66 text sizing protocol.
// This is a global setting that can be configured at startup.
var (
	kittyTextSizingEnabled = false
	kittyTextSizingMutex   sync.RWMutex
)

// SetKittyTextSizingEnabled enables or disables Kitty text sizing protocol support.
// When disabled, text will be rendered using standard ANSI escape sequences only.
func SetKittyTextSizingEnabled(enabled bool) {
	kittyTextSizingMutex.Lock()
	defer kittyTextSizingMutex.Unlock()
	kittyTextSizingEnabled = enabled
}

// IsKittyTextSizingEnabled returns whether Kitty text sizing is currently enabled.
func IsKittyTextSizingEnabled() bool {
	kittyTextSizingMutex.RLock()
	defer kittyTextSizingMutex.RUnlock()
	return kittyTextSizingEnabled
}

// hasKittyTextSizing returns true if the StylePrimitive has any Kitty text sizing properties set.
func hasKittyTextSizing(rules StylePrimitive) bool {
	return rules.KittyScale != nil ||
		rules.KittyWidth != nil ||
		rules.KittyNumerator != nil ||
		rules.KittyDenominator != nil
}

// buildKittyMetadata constructs the metadata string for OSC 66 escape sequence.
// Format: key=value pairs separated by colons
func buildKittyMetadata(rules StylePrimitive) string {
	var parts []string

	// s: scale (1-7), only include if > 1 (1 is default)
	if rules.KittyScale != nil && *rules.KittyScale > 1 && *rules.KittyScale <= 7 {
		parts = append(parts, fmt.Sprintf("s=%d", *rules.KittyScale))
	}

	// w: width (0-7), only include if explicitly set
	if rules.KittyWidth != nil && *rules.KittyWidth <= 7 {
		parts = append(parts, fmt.Sprintf("w=%d", *rules.KittyWidth))
	}

	// n: numerator (0-15) for fractional scaling
	if rules.KittyNumerator != nil && *rules.KittyNumerator <= 15 {
		parts = append(parts, fmt.Sprintf("n=%d", *rules.KittyNumerator))
	}

	// d: denominator (0-15), must be > n when non-zero
	if rules.KittyDenominator != nil && *rules.KittyDenominator <= 15 {
		parts = append(parts, fmt.Sprintf("d=%d", *rules.KittyDenominator))
	}

	// v: vertical alignment (0=top, 1=bottom, 2=centered)
	if rules.KittyVAlign != nil && *rules.KittyVAlign <= 2 {
		parts = append(parts, fmt.Sprintf("v=%d", *rules.KittyVAlign))
	}

	// h: horizontal alignment (0=left, 1=right, 2=centered)
	if rules.KittyHAlign != nil && *rules.KittyHAlign <= 2 {
		parts = append(parts, fmt.Sprintf("h=%d", *rules.KittyHAlign))
	}

	return strings.Join(parts, ":")
}

// applyANSIStyles applies standard ANSI styling (colors, bold, italic, etc.) to the text
// and returns the styled string.
func applyANSIStyles(p termenv.Profile, rules StylePrimitive, s string) string {
	out := termenv.String(s)

	if rules.Color != nil {
		out = out.Foreground(p.Color(*rules.Color))
	}
	if rules.BackgroundColor != nil {
		out = out.Background(p.Color(*rules.BackgroundColor))
	}
	if rules.Underline != nil && *rules.Underline {
		out = out.Underline()
	}
	if rules.Bold != nil && *rules.Bold {
		out = out.Bold()
	}
	if rules.Italic != nil && *rules.Italic {
		out = out.Italic()
	}
	if rules.CrossedOut != nil && *rules.CrossedOut {
		out = out.CrossOut()
	}
	if rules.Overlined != nil && *rules.Overlined {
		out = out.Overline()
	}
	if rules.Inverse != nil && *rules.Inverse {
		out = out.Reverse()
	}
	if rules.Blink != nil && *rules.Blink {
		out = out.Blink()
	}
	if rules.Faint != nil && *rules.Faint {
		out = out.Faint()
	}

	return out.String()
}

// renderKittyScaledText renders text using Kitty's OSC 66 text sizing protocol.
// ANSI styling is applied around the OSC 66 sequence since the protocol requires
// plain text inside the escape code.
//
// OSC 66 format: ESC ] 66 ; metadata ; text BEL
// Where:
//   - ESC ] is 0x1b 0x5d
//   - metadata is colon-separated key=value pairs
//   - BEL is 0x07
//
// When scale (s) > 1, the text occupies s×s cells per character.
// After rendering scaled text, additional newlines may be needed to account
// for the vertical space consumed.
func renderKittyScaledText(w io.Writer, p termenv.Profile, rules StylePrimitive, s string) {
	if len(s) == 0 {
		return
	}

	// Build the metadata string
	meta := buildKittyMetadata(rules)

	if meta == "" {
		// No kitty metadata, fall back to standard ANSI rendering
		styledText := applyANSIStyles(p, rules, s)
		_, _ = io.WriteString(w, styledText)
		return
	}

	// Build ANSI style prefix and suffix
	// We apply ANSI codes AROUND the OSC 66 sequence, not inside it
	prefix, suffix := buildANSIWrapper(p, rules)

	// DEBUG: Log the actual output
	if f, err := os.OpenFile("/tmp/osc66_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		output := fmt.Sprintf("%s\x1b]66;%s;%s\x07%s", prefix, meta, s, suffix)
		fmt.Fprintf(f, "DEBUG renderKittyScaledText: prefix=%q suffix=%q\n", prefix, suffix)
		fmt.Fprintf(f, "DEBUG full output bytes: %x\n", []byte(output))
		fmt.Fprintf(f, "DEBUG full output quoted: %q\n", output)
		f.Close()
	}

	// Write: ANSI_PREFIX + OSC66(plain_text) + ANSI_SUFFIX
	_, _ = io.WriteString(w, prefix)
	fmt.Fprintf(w, "\x1b]66;%s;%s\x07", meta, s)
	_, _ = io.WriteString(w, suffix)
}

// ansiEscapeRegex matches ANSI escape sequences
var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes all ANSI escape sequences from a string.
// This is useful for extracting plain text from styled content
// before wrapping it in OSC 66.
func StripANSI(s string) string {
	return ansiEscapeRegex.ReplaceAllString(s, "")
}

// buildANSIWrapper returns the ANSI escape sequence prefix and reset suffix
// for applying styles around content.
func buildANSIWrapper(p termenv.Profile, rules StylePrimitive) (prefix, suffix string) {
	// Build a styled empty string to extract the escape codes
	out := termenv.String("")

	if rules.Color != nil {
		out = out.Foreground(p.Color(*rules.Color))
	}
	if rules.BackgroundColor != nil {
		out = out.Background(p.Color(*rules.BackgroundColor))
	}
	if rules.Underline != nil && *rules.Underline {
		out = out.Underline()
	}
	if rules.Bold != nil && *rules.Bold {
		out = out.Bold()
	}
	if rules.Italic != nil && *rules.Italic {
		out = out.Italic()
	}
	if rules.CrossedOut != nil && *rules.CrossedOut {
		out = out.CrossOut()
	}
	if rules.Overlined != nil && *rules.Overlined {
		out = out.Overline()
	}
	if rules.Inverse != nil && *rules.Inverse {
		out = out.Reverse()
	}
	if rules.Blink != nil && *rules.Blink {
		out = out.Blink()
	}
	if rules.Faint != nil && *rules.Faint {
		out = out.Faint()
	}

	styled := out.String()
	// The styled string is just escape codes around empty content
	// Format is typically: \x1b[...m\x1b[0m
	// We want the opening codes as prefix and reset as suffix
	if len(styled) > 0 {
		// Find the reset sequence position
		resetSeq := "\x1b[0m"
		if idx := strings.Index(styled, resetSeq); idx >= 0 {
			prefix = styled[:idx]
			suffix = resetSeq
		}
	}

	return prefix, suffix
}

// GetKittyScaleRows returns the number of additional rows that scaled text will occupy.
// This is useful for cursor positioning after rendering scaled text.
// For scale s, text occupies s rows, so we need s-1 additional newlines after.
func GetKittyScaleRows(rules StylePrimitive) int {
	if rules.KittyScale != nil && *rules.KittyScale > 1 {
		return int(*rules.KittyScale) - 1
	}
	return 0
}

// KittyTextSizingCapability represents the level of OSC 66 support detected.
type KittyTextSizingCapability int

const (
	// KittyTextSizingNone - terminal does not support OSC 66
	KittyTextSizingNone KittyTextSizingCapability = iota
	// KittyTextSizingWidth - terminal supports width (w) parameter only
	KittyTextSizingWidth
	// KittyTextSizingFull - terminal supports full protocol including scale (s)
	KittyTextSizingFull
)

// cprResponseRegex matches CPR (Cursor Position Report) responses: ESC [ row ; col R
var cprResponseRegex = regexp.MustCompile(`\x1b\[(\d+);(\d+)R`)

// DetectKittyTextSizing queries the terminal to detect OSC 66 text sizing support.
// This uses the CPR (Cursor Position Report) method described in the Kitty protocol:
// 1. Send CR + CPR to get initial position
// 2. Send OSC 66 with w=2 (space in 2 cells) + CPR
// 3. Send OSC 66 with s=2 (space in 2x2 block) + CPR
// 4. Compare cursor positions to determine support level
//
// This function temporarily puts the terminal in raw mode and restores it afterward.
// It has a timeout to avoid blocking if the terminal doesn't respond.
//
// Returns the detected capability level and any error encountered.
func DetectKittyTextSizing(timeout time.Duration) (KittyTextSizingCapability, error) {
	// Check if we're running in a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return KittyTextSizingNone, nil
	}

	// Save terminal state and set raw mode
	oldState, err := makeRaw(os.Stdin.Fd())
	if err != nil {
		return KittyTextSizingNone, fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer restoreTerminal(os.Stdin.Fd(), oldState)

	// Create a buffered reader with timeout
	reader := bufio.NewReader(os.Stdin)

	// Helper to read CPR response with timeout
	readCPR := func() (row, col int, err error) {
		done := make(chan struct{})
		var response string

		go func() {
			buf := make([]byte, 32)
			for {
				n, readErr := reader.Read(buf)
				if readErr != nil {
					err = readErr
					close(done)
					return
				}
				response += string(buf[:n])
				if matches := cprResponseRegex.FindStringSubmatch(response); matches != nil {
					fmt.Sscanf(matches[1], "%d", &row)
					fmt.Sscanf(matches[2], "%d", &col)
					close(done)
					return
				}
			}
		}()

		select {
		case <-done:
			return row, col, err
		case <-time.After(timeout):
			return 0, 0, fmt.Errorf("timeout waiting for CPR response")
		}
	}

	// Move to start of line and query position
	fmt.Fprint(os.Stdout, "\r\x1b[6n") // CR + CPR request
	_, col1, err := readCPR()
	if err != nil {
		return KittyTextSizingNone, err
	}

	// Test width support: OSC 66 with w=2 draws space in 2 cells
	fmt.Fprint(os.Stdout, "\x1b]66;w=2; \x07\x1b[6n")
	_, col2, err := readCPR()
	if err != nil {
		return KittyTextSizingNone, err
	}

	// Test scale support: OSC 66 with s=2 draws space in 2x2 block
	fmt.Fprint(os.Stdout, "\x1b]66;s=2; \x07\x1b[6n")
	_, col3, err := readCPR()
	if err != nil {
		return KittyTextSizingNone, err
	}

	// Clear the test output
	fmt.Fprint(os.Stdout, "\r\x1b[K") // CR + clear to end of line

	// Analyze results
	// If col2 == col1, width isn't supported (terminal ignored OSC 66)
	// If col2 == col1 + 2, width is supported
	// If col3 == col2 + 2, scale is supported
	if col2 == col1 {
		return KittyTextSizingNone, nil
	}

	widthSupported := (col2 - col1) == 2
	scaleSupported := (col3 - col2) == 2

	if widthSupported && scaleSupported {
		return KittyTextSizingFull, nil
	} else if widthSupported {
		return KittyTextSizingWidth, nil
	}

	return KittyTextSizingNone, nil
}

// DetectAndEnableKittyTextSizing is a convenience function that detects support
// and enables the feature if full support is found.
// Returns the detected capability level.
func DetectAndEnableKittyTextSizing(timeout time.Duration) KittyTextSizingCapability {
	cap, err := DetectKittyTextSizing(timeout)
	if err != nil {
		return KittyTextSizingNone
	}
	if cap == KittyTextSizingFull {
		SetKittyTextSizingEnabled(true)
	}
	return cap
}

// IsKittyTerminal checks if we're likely running in Kitty by checking
// the TERM and TERM_PROGRAM environment variables.
// This is a quick heuristic check, not a definitive detection.
func IsKittyTerminal() bool {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	kittyPID := os.Getenv("KITTY_PID")

	return strings.Contains(term, "kitty") ||
		termProgram == "kitty" ||
		kittyPID != ""
}

// buildOSC66Output builds the complete OSC 66 output string with ANSI styling.
// This is used to create the content that will be base64 encoded to survive
// glamour's internal text processing.
func buildOSC66Output(p termenv.Profile, rules StylePrimitive, text string) string {
	meta := buildKittyMetadata(rules)
	if meta == "" {
		return applyANSIStyles(p, rules, text)
	}

	prefix, suffix := buildANSIWrapper(p, rules)
	return fmt.Sprintf("%s\x1b]66;%s;%s\x07%s", prefix, meta, text, suffix)
}

// base64Encode encodes a string to base64.
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// base64Decode decodes a base64 string.
func base64Decode(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecodeKittyTextSizeMarkers finds and decodes KITTY_TEXT_SIZE markers in text,
// replacing them with the actual OSC 66 sequences.
func DecodeKittyTextSizeMarkers(text string) string {
	markerPrefix := "KITTY_TEXT_SIZE:"
	markerSuffix := ":END_KITTY_TEXT_SIZE"

	result := text
	for {
		startIdx := strings.Index(result, markerPrefix)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(result[startIdx:], markerSuffix)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx

		// Extract the base64 content
		b64Start := startIdx + len(markerPrefix)
		b64Content := result[b64Start:endIdx]

		// Decode
		decoded, err := base64Decode(b64Content)
		if err != nil {
			// If decode fails, just remove the marker
			result = result[:startIdx] + result[endIdx+len(markerSuffix):]
			continue
		}

		// Replace marker with decoded content
		result = result[:startIdx] + decoded + result[endIdx+len(markerSuffix):]
	}

	return result
}
