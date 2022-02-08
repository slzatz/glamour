package ansi

import (
	"fmt"
	"io"
	"strconv"
)

// An ItemElement is used to render items inside a list.
type ItemElement struct {
	IsOrdered   bool
	Enumeration uint
}

func (e *ItemElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement
	if e.IsOrdered {
		el = &BaseElement{
			Style: ctx.options.Styles.Enumeration,
			//Prefix: strconv.FormatInt(int64(e.Enumeration), 10), //original
			Prefix: fmt.Sprintf("\x1b[1;36m%s. ", strconv.FormatInt(int64(e.Enumeration), 10)), // modified slz; quite a kluge
		}
	} else {
		el = &BaseElement{
			Style:  ctx.options.Styles.Item,
			Prefix: "\x1b[1;36m• ", // added slz
		}
	}

	return el.Render(w, ctx)
}
