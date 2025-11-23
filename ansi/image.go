package ansi

import (
	"fmt"
	"io"
	"strings"
)

// An ImageElement is used to render images elements.
type ImageElement struct {
	Text     string
	BaseURL  string
	URL      string
	Child    ElementRenderer
	TextOnly bool
}

// Render renders an ImageElement.
func (e *ImageElement) Render(w io.Writer, ctx RenderContext) error {
	// If kitty images are enabled, output kitty marker instead of standard image text
	if ctx.kittyImageConfig != nil && ctx.kittyImageConfig.Enabled && ctx.kittyImageConfig.ImageCache != nil {
		imageID, cols, rows, exists := ctx.kittyImageConfig.ImageCache(e.URL)
		if exists {
			// Output kitty image marker - will be replaced with Unicode placeholders later
			marker := fmt.Sprintf("[KITTY_IMAGE:id=%d,cols=%d,rows=%d]", imageID, cols, rows)
			_, err := w.Write([]byte(marker))
			return err
		}
		// If not in cache, fall through to standard rendering
	}

	// Standard image rendering (when kitty not enabled or image not in cache)
	style := ctx.options.Styles.ImageText
	if e.TextOnly {
		style.Format = strings.TrimSuffix(style.Format, " →")
	}

	if len(e.Text) > 0 {
		el := &BaseElement{
			Token: e.Text,
			Style: style,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	if e.TextOnly {
		return nil
	}

	if len(e.URL) > 0 {
		el := &BaseElement{
			Token:  resolveRelativeURL(e.BaseURL, e.URL),
			Prefix: " ",
			Style:  ctx.options.Styles.Image,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
