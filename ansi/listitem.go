package ansi

import (
	"io"
	"strconv"
  //"fmt" ////slz
)

// An ItemElement is used to render items inside a list.
type ItemElement struct {
	IsOrdered   bool
	Enumeration uint
}

// Render renders an ItemElement.
func (e *ItemElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement
	if e.IsOrdered {
		el = &BaseElement{
			Style:  ctx.options.Styles.Enumeration,
			Prefix: strconv.FormatInt(int64(e.Enumeration), 10), //nolint: gosec
		}
	} else {
  //fmt.Println("ItemElement.Render") ////slz
		el = &BaseElement{
			Style: ctx.options.Styles.Item,
			//Prefix: "+", ////slz
		}
	}

	return el.Render(w, ctx)
}
