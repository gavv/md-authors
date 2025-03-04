package gen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gavv/md-authors/src/defs"
)

type squashStep int

const (
	dontSquash squashStep = iota
	squashNonWsBefore
	squashWsBefore
	squashNonWsAfter
	squashWsAfter
)

// Format author to string according to conf.format template.
// Substitutes patterns like:
//   - {email}
//   - {email?}
//   - {email|profile}
//   - {email|profile?}
//
// ..with corresponding fields of author struct.
func formatAuthor(author defs.Author, conf defs.Config) (string, error) {
	fstr := conf.Format
	fpos := 0

	var (
		result           string
		curText          string
		curIsLiteral     bool
		trailingLiterals int
		squashState      squashStep = dontSquash
	)

	for fpos < len(fstr) {
		switch {
		case fstr[fpos] == '\\' && fpos < len(fstr)-1:
			// get escaped character like \n
			curText = fstr[fpos : fpos+2]
			curIsLiteral = true
			fpos += 2

			// unescape using go syntax rules
			rune, _, _, err := strconv.UnquoteChar(curText, 0)
			if err == nil {
				curText = string(rune)
			} else {
				// on error, copy second character as is
				curText = curText[1:]
			}

		case fstr[fpos] == '{':
			fpos++

			// find expression in curly braces
			end := fpos + 1
			for end < len(fstr) && fstr[end] != '}' {
				end++
			}
			if end == len(fstr) {
				return "", fmt.Errorf("bad format spec: missing trailing `}'")
			}

			// evaluate expression and squash flag
			field, squash, err := evalExpr(fstr[fpos:end], author)
			if err != nil {
				return "", err
			}
			fpos = end + 1

			curText = field
			curIsLiteral = false

			// decide whether we need to squash
			if curText != "" {
				squashState = dontSquash
			} else {
				if !squash {
					squashState = dontSquash
				} else {
					squashState = squashNonWsBefore
				}
			}

		default:
			// skip until curly brace
			end := fpos + 1
			for end < len(fstr) && fstr[end] != '\\' && fstr[end] != '{' {
				end++
			}

			// copy text as is
			curText = fstr[fpos:end]
			curIsLiteral = true
			fpos = end
		}

		// step 1: remove adjustent non-whitespaces before field
		if squashState == squashNonWsBefore {
			k := len(result)
			for k > 0 && trailingLiterals > 0 && !isSpace(result[k-1]) {
				k--
				trailingLiterals--
			}
			result = result[:k]
			squashState = squashWsBefore
		}

		// step 2: squash whitespaces before field into one
		if squashState == squashWsBefore {
			k := len(result)
			for k > 1 && isSpace(result[k-2]) {
				k--
			}
			trailingLiterals = 0
			result = result[:k]
			squashState = squashNonWsAfter
		}

		// step 3: remove adjustent non-whitespaces after field
		if squashState == squashNonWsAfter && len(curText) > 0 {
			if curIsLiteral {
				k := 0
				for k < len(curText) && !isSpace(curText[k]) {
					k++
				}
				curText = curText[k:]
				if len(curText) > 0 && isSpace(curText[0]) {
					squashState = squashWsAfter
				}
			} else {
				squashState = squashWsAfter
			}
		}

		// step 4: squash whitespaces before and after field into one
		if squashState == squashWsAfter && len(curText) > 0 {
			k := 0
			for k < len(curText) && isSpace(curText[k]) &&
				((k < len(curText)-1 && isSpace(curText[k+1])) ||
					(len(result) > 0 && isSpace(result[len(result)-1]))) {
				k++
			}
			curText = curText[k:]
			if len(curText) > 0 && !isSpace(curText[0]) {
				squashState = dontSquash
			}
		}

		// append text to result
		if len(curText) > 0 {
			result += curText
			if curIsLiteral {
				trailingLiterals += len(curText)
			} else {
				trailingLiterals = 0
			}
		}
	}

	// handle trailing whitespace if squash is in progress
	if squashState == squashWsAfter {
		k := len(result)
		for k > 0 && isSpace(result[k-1]) {
			k--
		}
		result = result[:k]
	}

	// if user haven't added newline, add it here
	if !strings.Contains(result, "\n") {
		result += "\n"
	}

	return result, nil
}

// Evaluates expression inside curly braces.
// See formatAuthor().
// Returns evaluated result and squash flag.
func evalExpr(expr string, author defs.Author) (string, bool, error) {
	squash := false
	if strings.HasSuffix(expr, "?") {
		squash = true
		expr = expr[:len(expr)-1]
	}

	field := ""
	for _, subexpr := range strings.Split(expr, "|") {
		var err error
		field, err = getField(subexpr, author)
		if err != nil {
			return "", false, err
		}
		if field != "" {
			break
		}
	}

	return field, squash, nil
}

// Get value of author's field by name.
func getField(name string, author defs.Author) (string, error) {
	result := ""

	switch name {
	case "index":
		result = fmt.Sprint(author.Index)
	case "date":
		result = author.Date
	case "name":
		result = author.Name
	case "email":
		result = author.Email
	case "login":
		result = author.Login
	case "profile":
		result = author.Profile
	default:
		return "", fmt.Errorf("bad format spec: unknown field `%s'", name)
	}

	return strings.TrimSpace(result), nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
