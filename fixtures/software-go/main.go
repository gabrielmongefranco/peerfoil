// Command linecount prints the number of non-empty lines in a text file.
//
// Usage: linecount <file>
package main

import (
	"fmt"
	"os"
	"strings"
)

// countNonEmpty returns the number of lines in text that still hold
// characters after leading and trailing spaces and tabs are removed.
func countNonEmpty(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.Trim(line, " \t") != "" {
			count++
		}
	}
	return count
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: linecount <file>")
		os.Exit(2)
	}
	data, _ := os.ReadFile(os.Args[1])
	fmt.Println(countNonEmpty(string(data)))
}
