package ansi

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/muesli/termenv"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// BaseElement renders a styled primitive element.
type BaseElement struct {
	Token  string
	Prefix string
	Suffix string
	Style  StylePrimitive
}

func formatToken(format string, token string) (string, error) {
	var b bytes.Buffer

	v := make(map[string]interface{})
	v["text"] = token

	tmpl, err := template.New(format).Funcs(TemplateFuncMap).Parse(format)
	if err != nil {
		return "", fmt.Errorf("glamour: error parsing template: %w", err)
	}

	err = tmpl.Execute(&b, v)
	return b.String(), err
}

func renderText(w io.Writer, p termenv.Profile, rules StylePrimitive, s string) {
	if len(s) == 0 {
		return
	}

	out := termenv.String(s)
	if rules.Upper != nil && *rules.Upper {
		out = termenv.String(cases.Upper(language.English).String(s))
	}
	if rules.Lower != nil && *rules.Lower {
		out = termenv.String(cases.Lower(language.English).String(s))
	}
	if rules.Title != nil && *rules.Title {
		out = termenv.String(cases.Title(language.English).String(s))
	}
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

	_, _ = io.WriteString(w, out.String())
}

// StyleOverrideRender renders a BaseElement with an overridden style.
func (e *BaseElement) StyleOverrideRender(w io.Writer, ctx RenderContext, style StylePrimitive) error {
	bs := ctx.blockStack
	st1 := cascadeStylePrimitives(bs.Current().Style.StylePrimitive, style)
	st2 := cascadeStylePrimitives(bs.With(e.Style), style)

	return e.doRender(w, ctx.options.ColorProfile, st1, st2)
}

// Render renders a BaseElement.
func (e *BaseElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	st1 := bs.Current().Style.StylePrimitive
	st2 := bs.With(e.Style)
	return e.doRender(w, ctx.options.ColorProfile, st1, st2)
}

func (e *BaseElement) doRender(w io.Writer, p termenv.Profile, st1, st2 StylePrimitive) error {
	
	prefixStyle := st1 // Use the element style as the base st1 is the parent
	prefixStyle.Color = st1.PrefixColor
	renderText(w, p, prefixStyle, e.Prefix)
// I don't see any use of autogenerated suffix in the code
//	defer func() {
//		renderText(w, p, prefixStyle, e.Suffix)
//	}()

  //renders the block_prefix of elements like (e.g., ordered and unordered lists, definition lists) in the style 
  // of the parent (e.g., list, definition_list) using the prefix_color of the parent to color the block_prefix
  blockPrefixStyle := st1 // Use the element style of the parent
  blockPrefixStyle.Color = st1.PrefixColor
  renderText(w, p, blockPrefixStyle, st2.BlockPrefix)
	defer func() {
		renderText(w, p, blockPrefixStyle, st2.BlockSuffix)
	}()

	// render styled prefix/suffix
	renderText(w, p, st2, st2.Prefix)
	
	defer func() {
		renderText(w, p, st2, st2.Suffix)
	}()

	// Format and render the token content
	s := e.Token
	if len(st2.Format) > 0 {
		var err error
		s, err = formatToken(st2.Format, s)
		if err != nil {
			return err
		}
	}
	
	renderText(w, p, st2, escapeReplacer.Replace(s))
	return nil
}

// https://www.markdownguide.org/basic-syntax/#characters-you-can-escape
var escapeReplacer = strings.NewReplacer(
	"\\\\", "\\",
	"\\`", "`",
	"\\*", "*",
	"\\_", "_",
	"\\{", "{",
	"\\}", "}",
	"\\[", "[",
	"\\]", "]",
	"\\<", "<",
	"\\>", ">",
	"\\(", "(",
	"\\)", ")",
	"\\#", "#",
	"\\+", "+",
	"\\-", "-",
	"\\.", ".",
	"\\!", "!",
	"\\|", "|",
)
