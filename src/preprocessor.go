package main

import (
	"bufio"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

type PreprocessError struct {
	text    string
	line    int
	colFrom int
	colTo   int
}

type Preprocessor struct {
	writer         *bufio.Writer
	source         string
	scopeStack     []bool
	variables      map[string]string
	versions       map[string]int
	errors         []PreprocessError
	line           int
	currentVersion int
	invalid        bool
}

func unCR(str string) string {
	if len(str) > 0 && str[len(str)-1] == '\r' {
		return str[:len(str)-1]
	}
	return str
}

func emitError(p *Preprocessor, err PreprocessError) {
	p.invalid = true
	p.errors = append(p.errors, err)
}

func pushStack(p *Preprocessor, val bool) {
	p.scopeStack = append(p.scopeStack, p.scopeStack[len(p.scopeStack)-1] && val)
}

func PreprocessString(src string, output *bufio.Writer, variables map[string]string, versions map[string]int, currentVersion int) Preprocessor {
	p := Preprocessor{source: src, writer: output, scopeStack: []bool{true}, line: 1, variables: variables, versions: versions, currentVersion: currentVersion}

	for l := range strings.SplitSeq(p.source, "\n") {
		if acceptControl(l) {
			if strings.HasPrefix(l[3:], "if") {
				word1, r1, ok := nextWord(l, 5)
				// fmt.Printf("'%s' %v\n", word, ok)
				word2, r2, ok := nextWord(l, r1)
				// fmt.Printf("'%s' %v\n", word, ok)
				if ok {
					var word3 string
					word3, r3, ok := nextWord(l, r2)
					if !ok {
						pushStack(&p, false)
						emitError(&p, PreprocessError{"unrecognized instruction", p.line, r1 - len(word1), r3})
					} else {
						pushStack(&p, handleToExpr(&p, word1, word2, word3, r1, r2, r3))
					}
				} else {
					res := p.variables[word1] != ""
					pushStack(&p, res)
				}
				//fmt.Printf("[%s] [%s] %v\n", word, r, ok)
			} else if strings.HasPrefix(l[3:], "endif") {
				if len(p.scopeStack) == 1 {
					emitError(&p, PreprocessError{"@endif closes non-existent if", p.line, 1, len(l)})
				} else {
					p.scopeStack = p.scopeStack[:len(p.scopeStack)-1]
				}
			}
		} else {
			fmt.Printf("%-20s | %s\n", fmt.Sprintf("%v", p.scopeStack), l)
			if !p.invalid && p.scopeStack[len(p.scopeStack)-1] {
				p.writer.WriteString(l)
				p.writer.WriteString("\n")
			}
		}
		p.line += 1
	}

	if len(p.scopeStack) != 1 {
		emitError(&p, PreprocessError{"one or more @if scopes haven't been closed", p.line - 1, 1, 2})
	}

	return p
}

func acceptControl(str string) bool {
	return strings.HasPrefix(str, "//@")
}

func nextWord(str string, cur int) (word string, remainder int, ok bool) {
	for {
		r, n := utf8.DecodeRuneInString(str[cur:])
		if r == utf8.RuneError {
			return
		}

		if unicode.IsSpace(r) {
			cur += n
		} else {
			break
		}
	}

	start := cur
	for {
		r, n := utf8.DecodeRuneInString(str[cur:])
		if r == utf8.RuneError {
			return
		}

		if !unicode.IsSpace(r) {
			cur += n
		} else {
			break
		}
	}

	return str[start:cur], cur, true
}

func getVersion(p *Preprocessor, word string, r, noBound, failed int) (version int) {
	if word == "..." {
		return noBound
	}
	version, ok := p.versions[word]
	if !ok {
		emitError(p, PreprocessError{"version doesn't exist", p.line, r - len(word), r})
		return failed
	}
	return version
}

func handleToExpr(p *Preprocessor, word1, word2, word3 string, r1, r2, r3 int) (ok bool) {
	fromNotEqual := strings.HasPrefix(word2, "<")
	toNotEqual := strings.HasSuffix(word2, "<")
	if fromNotEqual {
		word2 = word2[1:]
	}

	if toNotEqual {
		word2 = word2[:len(word2)-1]
	}

	if word2 != "to" {
		emitError(p, PreprocessError{"unrecognized instruction", p.line, r1 - len(word1), r3})
		return
	}

	fromVersion := getVersion(p, word1, r1, math.MinInt, math.MaxInt)
	toVersion := getVersion(p, word3, r3, math.MaxInt, math.MinInt)

	fmt.Println(fromVersion, p.currentVersion, toVersion)

	var res bool
	if fromNotEqual {
		res = fromVersion < p.currentVersion
	} else {
		res = fromVersion <= p.currentVersion
	}

	if toNotEqual {
		res = res && p.currentVersion < toVersion
	} else {
		res = res && p.currentVersion <= toVersion
	}

	return res
}
