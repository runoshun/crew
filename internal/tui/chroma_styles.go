package tui

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
)

func init() {
	// Register Catppuccin Mocha style
	// Based on https://github.com/catppuccin/chroma
	styles.Register(chroma.MustNewStyle("catppuccin-mocha", chroma.StyleEntries{
		chroma.Text:                "#cdd6f4",
		chroma.Error:               "#f38ba8",
		chroma.Comment:             "#6c7086 italic",
		chroma.CommentPreproc:      "#f5e0dc",
		chroma.Keyword:             "#cba6f7",
		chroma.KeywordType:         "#f9e2af",
		chroma.KeywordDeclaration:  "#cba6f7 italic",
		chroma.KeywordNamespace:    "#f5c2e7",
		chroma.Operator:            "#89dceb",
		chroma.Punctuation:         "#9399b2",
		chroma.Name:                "#cdd6f4",
		chroma.NameAttribute:       "#f9e2af",
		chroma.NameClass:           "#f9e2af",
		chroma.NameConstant:        "#fab387",
		chroma.NameDecorator:       "#f5c2e7",
		chroma.NameFunction:        "#89b4fa",
		chroma.NameTag:             "#cba6f7",
		chroma.Literal:             "#cdd6f4",
		chroma.LiteralNumber:       "#fab387",
		chroma.LiteralString:       "#a6e3a1",
		chroma.LiteralStringEscape: "#f5e0dc",
		chroma.GenericDeleted:      "#f38ba8",
		chroma.GenericEmph:         "italic",
		chroma.GenericHeading:      "#89b4fa bold",
		chroma.GenericInserted:     "#a6e3a1",
		chroma.GenericStrong:       "bold",
		chroma.GenericSubheading:   "#a6adc8 bold",
		chroma.Background:          "", // Transparent background
	}))
}
