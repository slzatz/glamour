package ansi

import (
	"fmt"
	"io"
	"net/url"
)

//var NumLinks int

// A LinkElement is used to render hyperlinks.
type LinkElement struct {
	Text    string
	BaseURL string
	URL     string
	Child   ElementRenderer // FIXME
}

func (e *LinkElement) Render(w io.Writer, ctx RenderContext) error {
	var textRendered bool
	var token string
	//if len(e.Text) > 0 && e.Text != e.URL {
	if ctx.options.LinkNumbers {
		textRendered = true
		NumLinks++
		token = fmt.Sprintf("[%d]", NumLinks)
	}

	el := &BaseElement{
		//Token: e.Text,
		Token: token,
		Style: ctx.options.Styles.LinkText,
	}
	err := el.Render(w, ctx)
	if err != nil {
		return err
	}

	/*
		if node.LastChild != nil {
			if node.LastChild.Type == bf.Image {
				el := tr.NewElement(node.LastChild)
				err := el.Renderer.Render(w, node.LastChild, tr)
				if err != nil {
					return err
				}
			}
			if len(node.LastChild.Literal) > 0 &&
				string(node.LastChild.Literal) != string(node.LinkData.Destination) {
				textRendered = true
				el := &BaseElement{
					Token: string(node.LastChild.Literal),
					Style: ctx.style[LinkText],
				}
				err := el.Render(w, node.LastChild, tr)
				if err != nil {
					return err
				}
			}
		}
	*/

	u, err := url.Parse(e.URL)
	if err == nil &&
		"#"+u.Fragment != e.URL { // if the URL only consists of an anchor, ignore it
		pre := " "
		style := ctx.options.Styles.Link
		if !textRendered {
			pre = ""
			style.BlockPrefix = ""
			style.BlockSuffix = ""
		}

		el := &BaseElement{
			Token:  "\x1b]8;;" + resolveRelativeURL(e.BaseURL, e.URL) + "\x1b\\" + e.Text + "\x1b]8;;\x1b\\",
			Prefix: pre,
			Style:  style,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
