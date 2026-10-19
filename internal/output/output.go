// Package output renders command results in the family's text formats
// (cli-common output-and-rendering.md §4).
package output

import (
	"fmt"
	"io"
	"strings"
)

var cellCleaner = strings.NewReplacer(" | ", " ", "\r\n", " ", "\n", " ", "\r", " ")

// Table writes a pipe-delimited table. Empty cells render as "-"; an
// embedded " | " or newline in a cell becomes a single space.
func Table(w io.Writer, headers []string, rows [][]string) {
	fmt.Fprintln(w, strings.Join(headers, " | "))
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, c := range row {
			c = strings.TrimSpace(cellCleaner.Replace(c))
			if c == "" {
				c = "-"
			}
			cells[i] = c
		}
		fmt.Fprintln(w, strings.Join(cells, " | "))
	}
}
