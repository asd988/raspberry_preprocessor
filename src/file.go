package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func PrettyStringError(err PreprocessError, src, path string) string {
	var sb strings.Builder
	path, _ = filepath.Abs(path)
	fmt.Fprintf(&sb, "%s:%d:%d:\n%s error: %s\n", path, err.line, err.colFrom+1, "preprocess", err.text)
	prefix := fmt.Sprintf("%2d | ", err.line)
	cur := 1

	sb.WriteString(prefix)
	for v := range strings.SplitSeq(src, "\n") {
		if cur == err.line {
			sb.WriteString(v)
			break
		}
		cur += 1
	}
	sb.WriteRune('\n')

	for i := 0; i < len(prefix)+err.colFrom; i++ {
		sb.WriteRune(' ')
	}
	for i := 0; i < err.colTo-err.colFrom; i++ {
		sb.WriteString("^")
	}

	sb.WriteString(" here\n\n")

	return sb.String()
}

func preprocessFile(sourcePath, outputPath string, variables map[string]string, versions map[string]int, currentVersion int) bool {
	dataSource, _ := os.ReadFile(sourcePath)
	source := string(dataSource)

	var p Preprocessor
	{
		f, _ := os.Create(outputPath)

		buf := bufio.NewWriter(f)

		p = PreprocessString(source, buf, variables, versions, currentVersion)
		for _, v := range p.errors {
			fmt.Print(PrettyStringError(v, source, sourcePath))
		}

		buf.Flush()
		f.Close()
	}

	if p.invalid || p.exclude {
		os.Remove(outputPath)
	}

	return !p.invalid
}
