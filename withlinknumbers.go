package glamour

// Added by slz 06242021
func WithLinkNumbers(linkNumbers bool) TermRendererOption {
	return func(tr *TermRenderer) error {
		tr.ansiOptions.LinkNumbers = linkNumbers
		return nil
	}
}
