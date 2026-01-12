package main

import (
	"bufio"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Scope int

const (
	ScopeActive = (1 << iota)
	ScopeBeenActive
	ScopeElseUsed
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
	scopeStack     []Scope
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

func scopeFromEnd(p *Preprocessor, n int) *Scope {
	return &p.scopeStack[len(p.scopeStack)-1-n]
}

func isActive(p *Preprocessor, n int) bool {
	return *scopeFromEnd(p, n)&ScopeActive != 0
}

func beenActive(p *Preprocessor, n int) bool {
	return *scopeFromEnd(p, n)&ScopeBeenActive != 0
}

func isElseUsed(p *Preprocessor, n int) bool {
	return *scopeFromEnd(p, n)&ScopeElseUsed != 0
}

func pushStack(p *Preprocessor, val bool) {
	if p.scopeStack[len(p.scopeStack)-1]&ScopeActive != 0 && val {
		p.scopeStack = append(p.scopeStack, ScopeActive|ScopeBeenActive)
	} else {
		p.scopeStack = append(p.scopeStack, 0)
	}
}

func PreprocessString(src string, output *bufio.Writer, variables map[string]string, versions map[string]int, currentVersion int) Preprocessor {
	p := Preprocessor{source: src, writer: output, scopeStack: []Scope{ScopeActive}, line: 1, variables: variables, versions: versions, currentVersion: currentVersion}

	for l := range strings.SplitSeq(p.source, "\n") {
		if acceptControl(l) {
			if strings.HasPrefix(l[3:], "if") {
				pushStack(&p, handleIf(&p, l, 5))
			} else if strings.HasPrefix(l[3:], "elif") {
				res := handleIf(&p, l, 7)
				if len(p.scopeStack) < 2 {
					emitError(&p, PreprocessError{"@elif not inside @if scope", p.line, 1, len(l)})
				} else if isElseUsed(&p, 0) {
					emitError(&p, PreprocessError{"@elif can't go after an @else", p.line, 1, len(l)})
				} else if isActive(&p, 1) {
					end := scopeFromEnd(&p, 0)
					val := *end
					if beenActive(&p, 0) {
						*end = val & ^ScopeActive
					} else if res {
						*end = val | ScopeActive | ScopeBeenActive
					}
				}
			} else if strings.HasPrefix(l[3:], "else") {
				_, _, ok := nextWord(l, 7)
				if ok {
					emitError(&p, PreprocessError{"@else mustn't have any operands", p.line, 1, len(l)})
				}

				if len(p.scopeStack) < 2 {
					emitError(&p, PreprocessError{"@else not inside @if scope", p.line, 1, len(l)})
				} else if isElseUsed(&p, 0) {
					emitError(&p, PreprocessError{"@else can be used once", p.line, 1, len(l)})
				} else if isActive(&p, 1) {
					end := scopeFromEnd(&p, 0)
					val := *end
					if beenActive(&p, 0) {
						*end = val & ^ScopeActive
					} else {
						*end = val | ScopeActive | ScopeBeenActive
					}
				}
			} else if strings.HasPrefix(l[3:], "endif") {
				if len(p.scopeStack) < 2 {
					emitError(&p, PreprocessError{"@endif closes non-existent @if scope", p.line, 1, len(l)})
				} else {
					p.scopeStack = p.scopeStack[:len(p.scopeStack)-1]
				}
			}
		} else {
			fmt.Printf("%-20s | %s\n", fmt.Sprintf("%v", p.scopeStack), l)
			if !p.invalid && isActive(&p, 0) {
				p.writer.WriteString(substitute(&p, l))
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

func handleIf(p *Preprocessor, l string, start int) bool {
	word1, r1, ok := nextWord(l, start)
	// fmt.Printf("'%s' %v\n", word, ok)
	word2, r2, ok := nextWord(l, r1)
	// fmt.Printf("'%s' %v\n", word, ok)
	if ok {
		var word3 string
		word3, r3, ok := nextWord(l, r2)
		if !ok {
			emitError(p, PreprocessError{"unrecognized instruction", p.line, r1 - len(word1), r3})
			return false
		} else {
			return handleToExpr(p, word1, word2, word3, r1, r2, r3)
		}
	}
	res := p.variables[word1] != "" || p.versions[word1] != 0
	return res
}

func substitute(p *Preprocessor, l string) string {
	entire := l
	type ReplaceRange struct {
		from, to int
	}
	var inserts []ReplaceRange

	prev := 0

	for {
		ix := strings.Index(l, "@@(")
		if ix == -1 {
			break
		}

		ix += 3
		start := ix
		incorrect := false
		for {
			r, n := utf8.DecodeRuneInString(l[ix:])

			if r == utf8.RuneError {
				emitError(p, PreprocessError{"the insert macro isn't closed", p.line, prev + start, prev + ix})
				break
			}

			ix += n

			if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '_' {

			} else if r == ')' {
				break
			} else {
				incorrect = true
			}
		}

		if incorrect {
			emitError(p, PreprocessError{"the insert macro's body must consist of alphanumeric characters or _", p.line, prev + start, prev + ix})
		} else {
			inserts = append(inserts, ReplaceRange{prev + start - 3, prev + ix})
		}

		prev += ix
		l = l[ix:]
	}

	var sb strings.Builder
	prev = 0
	for _, v := range inserts {
		id := entire[v.from+3 : v.to-1]
		insertStr, ok := p.variables[id]
		if !ok {
			emitError(p, PreprocessError{"no variable found with this name", p.line, v.from, v.to})
			continue
		}

		sb.WriteString(entire[prev:v.from])
		sb.WriteString(insertStr)
		prev = v.to
	}
	sb.WriteString(entire[prev:])

	return sb.String()
}
