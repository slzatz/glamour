package ansi

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/muesli/reflow/wordwrap"
)

// A HeadingElement is used to render headings.
type HeadingElement struct {
	Level int
	First bool
}

const (
	h1 = iota + 1
	h2
	h3
	h4
	h5
	h6
)

// Render renders a HeadingElement.
func (e *HeadingElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.options.Styles.Heading

	switch e.Level {
	case h1:
		rules = cascadeStyles(rules, ctx.options.Styles.H1)
	case h2:
		rules = cascadeStyles(rules, ctx.options.Styles.H2)
	case h3:
		rules = cascadeStyles(rules, ctx.options.Styles.H3)
	case h4:
		rules = cascadeStyles(rules, ctx.options.Styles.H4)
	case h5:
		rules = cascadeStyles(rules, ctx.options.Styles.H5)
	case h6:
		rules = cascadeStyles(rules, ctx.options.Styles.H6)
	}

	if !e.First {
		renderText(w, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, "\n")
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(bs.Current().Style, rules, false),
	}
	bs.Push(be)

	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockPrefix)
	renderText(bs.Current().Block, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, rules.Prefix)
	return nil
}

// Finish finishes rendering a HeadingElement.
func (e *HeadingElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := bs.Current().Style

	// Check if Kitty text sizing is enabled and this heading style has scale set
	if IsKittyTextSizingEnabled() && hasKittyTextSizing(rules.StylePrimitive) {
		// Extract plain text from the block buffer (strip any per-token ANSI codes)
		blockContent := bs.Current().Block.String()
		plainText := StripANSI(blockContent)

		// Include the suffix in the scaled text so it matches the heading height
		// The suffix is typically a trailing space with background color
		fullText := plainText + rules.Suffix

		// DEBUG: Log to file
		if f, err := os.OpenFile("/tmp/osc66_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			scaleVal := uint(0)
			if rules.StylePrimitive.KittyScale != nil {
				scaleVal = *rules.StylePrimitive.KittyScale
			}
			meta := buildKittyMetadata(rules.StylePrimitive)
			fmt.Fprintf(f, "DEBUG OSC66: scale=%d meta=%q fullText=%q\n", scaleVal, meta, fullText)
			f.Close()
		}

		// IMPORTANT: We need to write OSC 66 content with a special marker that
		// will survive glamour's internal processing (BlockElement, MarginWriter, etc.)
		// The marker format: KITTY_TEXT_SIZE:base64(osc66_sequence):END_KITTY_TEXT_SIZE
		// This will be decoded later in the rendering pipeline.
		osc66Output := buildOSC66Output(ctx.options.ColorProfile, rules.StylePrimitive, fullText)
		marker := fmt.Sprintf("KITTY_TEXT_SIZE:%s:END_KITTY_TEXT_SIZE", base64Encode(osc66Output))
		_, _ = io.WriteString(w, marker)

		// Render block suffix only (suffix is now part of the scaled text)
		renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockSuffix)

		// Add extra newlines for scaled text vertical space
		extraRows := GetKittyScaleRows(rules.StylePrimitive)
		for i := 0; i < extraRows; i++ {
			_, _ = io.WriteString(w, "\n")
		}

		bs.Current().Block.Reset()
		bs.Pop()
		return nil
	}

	// Standard rendering path (no Kitty text sizing)
	mw := NewMarginWriter(ctx, w, rules)

	flow := wordwrap.NewWriter(int(bs.Width(ctx))) //nolint: gosec
	_, err := flow.Write(bs.Current().Block.Bytes())
	if err != nil {
		return fmt.Errorf("glamour: error writing bytes: %w", err)
	}
	if err := flow.Close(); err != nil {
		return fmt.Errorf("glamour: error closing flow: %w", err)
	}

	_, err = mw.Write(flow.Bytes())
	if err != nil {
		return err
	}

	renderText(w, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, rules.Suffix)
	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockSuffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
