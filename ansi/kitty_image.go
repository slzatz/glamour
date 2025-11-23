package ansi

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// KittyImageConfig holds configuration for kitty image rendering
type KittyImageConfig struct {
	Enabled      bool
	ImageCache   func(string) (uint32, int, int, bool) // url -> (imageID, cols, rows, exists)
}

// KittyImageRenderer intercepts markdown image nodes and emits Unicode placeholders
// for images that have been pre-transmitted to kitty
type KittyImageRenderer struct {
	config KittyImageConfig
}

// NewKittyImageRenderer creates a new kitty image renderer
func NewKittyImageRenderer(config KittyImageConfig) renderer.NodeRenderer {
	return &KittyImageRenderer{config: config}
}

// RegisterFuncs registers the renderer for markdown image nodes
func (r *KittyImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

// renderImage outputs Unicode placeholders for pre-transmitted kitty images
// For relative placements: outputs a 1x1 transparent placeholder that flows with text
func (r *KittyImageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	// If kitty not enabled, let default renderer handle it
	if !r.config.Enabled || r.config.ImageCache == nil {
		return ast.WalkContinue, nil
	}

	img := node.(*ast.Image)
	url := string(img.Destination)

	// Look up the pre-transmitted image in cache (this validates the image exists)
	imageID, cols, rows, exists := r.config.ImageCache(url)
	if !exists {
		// Image wasn't pre-transmitted, let default renderer handle it
		w.WriteString(fmt.Sprintf("[IMAGE NOT IN CACHE: %s]", url))
		return ast.WalkContinue, nil
	}

	// Output a simple marker that we'll use to create placements later
	// Format: [KITTY_IMAGE:id=X,cols=Y,rows=Z]
	w.WriteString(fmt.Sprintf("[KITTY_IMAGE:id=%d,cols=%d,rows=%d]", imageID, cols, rows))

	// Skip children to prevent default image rendering
	return ast.WalkSkipChildren, nil
}

// Diacritics table for encoding row/column positions (32 available)
var kittyDiacritics = []rune{
	'\u0305', '\u030D', '\u030E', '\u0310', '\u0312', '\u033D', '\u033E', '\u033F',
	'\u0346', '\u034A', '\u034B', '\u034C', '\u0350', '\u0351', '\u0352', '\u0357',
	'\u035B', '\u0363', '\u0364', '\u0365', '\u0366', '\u0367', '\u0368', '\u0369',
	'\u036A', '\u036B', '\u036C', '\u036D', '\u036E', '\u036F', '\u0483', '\u0484',
}

// kittyRowColDiacritic returns the diacritic for a given row or column index
func kittyRowColDiacritic(idx int) string {
	if idx < 0 || idx >= len(kittyDiacritics) {
		return ""
	}
	return string(kittyDiacritics[idx])
}

// buildTransparentPlaceholderGrid builds a 1x1 Unicode placeholder for the transparent image
// This placeholder flows with text, and actual images are positioned relative to it
func buildTransparentPlaceholderGrid() string {
	const PLACEHOLDER_IMAGE_ID = 1     // Transparent 1x1 image ID
	const PLACEHOLDER_PLACEMENT_ID = 1 // Transparent placeholder's placement ID

	// Encode image ID in foreground color (38;2)
	red := (PLACEHOLDER_IMAGE_ID >> 16) & 0xff
	green := (PLACEHOLDER_IMAGE_ID >> 8) & 0xff
	blue := PLACEHOLDER_IMAGE_ID & 0xff
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", red, green, blue)

	// Encode placement ID in underline color (58;2) - CRITICAL for kitty to recognize reference
	ulRed := (PLACEHOLDER_PLACEMENT_ID >> 16) & 0xff
	ulGreen := (PLACEHOLDER_PLACEMENT_ID >> 8) & 0xff
	ulBlue := PLACEHOLDER_PLACEMENT_ID & 0xff
	ul := fmt.Sprintf("\x1b[58;2;%d;%d;%dm", ulRed, ulGreen, ulBlue)

	reset := "\x1b[0m"
	placeholderRune := "\U00010EEEE"

	// Output single 1x1 placeholder (row 0, col 0)
	rowDia := kittyRowColDiacritic(0)
	colDia := kittyRowColDiacritic(0)

	return fg + ul + placeholderRune + rowDia + colDia + reset
}

// buildKittyPlaceholderGrid builds Unicode placeholder grid for a pre-transmitted image
// For regular placements (a=T), placement ID = image ID
func buildKittyPlaceholderGrid(imageID uint32, cols, rows int) string {
	// Encode image ID in foreground color (38;2)
	imgRed := (imageID >> 16) & 0xff
	imgGreen := (imageID >> 8) & 0xff
	imgBlue := imageID & 0xff
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", imgRed, imgGreen, imgBlue)

	// Encode placement ID in underline color (58;2) - same as image ID for a=T
	placementID := imageID
	plRed := (placementID >> 16) & 0xff
	plGreen := (placementID >> 8) & 0xff
	plBlue := placementID & 0xff
	ul := fmt.Sprintf("\x1b[58;2;%d;%d;%dm", plRed, plGreen, plBlue)

	reset := "\x1b[0m"
	placeholderRune := "\U00010EEEE"

	var sb strings.Builder
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			rowDia := kittyRowColDiacritic(row)
			colDia := kittyRowColDiacritic(col)
			// Each cell: foreground + underline + placeholder + diacritics
			sb.WriteString(fg)
			sb.WriteString(ul)
			sb.WriteString(placeholderRune)
			sb.WriteString(rowDia)
			sb.WriteString(colDia)
		}
		// Newline at end of each row
		sb.WriteString("\n")
	}
	// Reset all attributes at end
	sb.WriteString(reset)
	return sb.String()
}
