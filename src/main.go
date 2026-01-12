package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

func PrettyStringError(err PreprocessError, src string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "src:%d:%d: %s error: %s\n", err.line, err.colFrom, "preprocess", err.text)
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

type Item struct {
	Id          string `json:"id"`
	DataVersion int    `json:"data_version"`
}

func main() {
	data, err := os.ReadFile(".rpb_cache/versions.json")
	if err != nil {
		return
	}

	var items []Item
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		log.Fatal(err)
	}

	m := make(map[string]int, len(items))
	for _, item := range items {
		m[item.Id] = item.DataVersion
	}

	dataSource, _ := os.ReadFile("./examples/00_Start/dummy.txt")
	source := string(dataSource)

	f, _ := os.Create("output.txt")
	defer f.Close()

	buf := bufio.NewWriter(f)
	defer buf.Flush()

	p := PreprocessString(source, buf, nil, m, m["1.20.1"])
	for _, v := range p.errors {
		fmt.Print(PrettyStringError(v, source))
	}

}
