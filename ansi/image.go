package ansi

import (
	//	"bufio"
	"fmt"
	"github.com/slzatz/glamour/kitty"
	"io"
	"log"
	"os"
)

// An ImageElement is used to render images elements.
type ImageElement struct {
	Text    string
	BaseURL string
	URL     string
	Child   ElementRenderer // FIXME
}

func (e *ImageElement) Render(w io.Writer, ctx RenderContext) error {
	if len(e.Text) > 0 {
		el := &BaseElement{
			Token: e.Text,
			Style: ctx.options.Styles.ImageText,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
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
		/*
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("\x1b[6n")
			text, _ := reader.ReadLine()
		*/

		//fmt.Printf("\x1b[%d;%dH", 20, 100)
		fmt.Println("Hello World")
		iImg, _, err := kitty.loadImage("/home/slzatz/Pictures/wood_ducks_smaller.jpg")
		if err != nil {
			log.Fatal(err)
		}
		err = kitty.KittyWriteImage(os.Stdout, iImg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("")
		//fmt.Println(text)
	}

	return nil
}
