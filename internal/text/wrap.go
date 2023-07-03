package text

import "strings"

func WrapAndIndent(input, prefix string, width int) string {
	// Split input into paragraphs
	paragraphs := strings.Split(input, "\n\n")
	var wrappedParagraphs []string

	// Wrap and indent each paragraph
	for _, paragraph := range paragraphs {
		lines := strings.Split(paragraph, "\n")
		var wrappedLines []string
		for _, line := range lines {
			wrappedLines = append(wrappedLines, Wrap(line, width))
		}
		indentedLines := Indent(strings.Join(wrappedLines, "\n"), prefix)
		wrappedParagraphs = append(wrappedParagraphs, indentedLines)
	}

	return strings.Join(wrappedParagraphs, "\n\n")
}

func Wrap(input string, width int) string {
	words := strings.Fields(input)
	var lines []string
	currentLine := ""
	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			lines = append(lines, currentLine)
			currentLine = ""
		}
		if currentLine == "" {
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return strings.Join(lines, "\n")
}

func Indent(input, prefix string) string {
	lines := strings.Split(input, "\n")
	indentedLines := make([]string, len(lines))
	for i, line := range lines {
		indentedLines[i] = prefix + line
	}
	return strings.Join(indentedLines, "\n")
}
