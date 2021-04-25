package ansi

import (
	"fmt"
	"io"
	"strconv"
)

// An ItemElement is used to render items inside a list.
type ItemElement struct {
	Enumeration uint
}

func (e *ItemElement) Render(w io.Writer, ctx RenderContext) error {
	var el *BaseElement
	if e.Enumeration > 0 {
		el = &BaseElement{
			Style: ctx.options.Styles.Enumeration,
			//Prefix: "\x1b[36m" + strconv.FormatInt(int64(e.Enumeration), 10) + ". ", // modified slz; quite a kluge
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
