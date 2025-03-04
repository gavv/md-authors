package gen

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gavv/md-authors/src/backend"
	"github.com/gavv/md-authors/src/defs"
	"github.com/gavv/md-authors/src/logs"
	"github.com/gavv/md-authors/src/match"
)

var (
	emptyLinesRx = regexp.MustCompile(`\n(\s*\n+)+`)
	spaceRx      = regexp.MustCompile(`\s`)
)

func regenerateBlock(content string, conf defs.Config) (string, error) {
	content = strings.Trim(content, "\n")
	if content != "" {
		content += "\n"
	}

	content, err := generateAuthors(content, conf)
	if err != nil {
		return "", err
	}

	content = strings.Trim(content, "\n")
	if content != "" {
		content = "\n" + content + "\n\n"
	}

	return content, nil
}

func generateAuthors(content string, conf defs.Config) (string, error) {
	allAuthors, err := backend.CollectAuthors(conf)
	if err != nil {
		return "", err
	}

	if conf.Sort == "name" {
		sort.SliceStable(allAuthors, func(i, j int) bool {
			return sortKey(allAuthors[i]) < sortKey(allAuthors[j])
		})
	}

	var (
		index int
		added int
	)

	if conf.Append {
		// Assume that 1 author = 1 non-blank line.
		stripped := emptyLinesRx.ReplaceAllString(content, "\n")
		stripped = strings.TrimSpace(stripped)
		if stripped != "" {
			stripped += "\n"
		}
		index = strings.Count(stripped, "\n")
	} else {
		content = ""
	}

	seenAuthors := make(map[string]struct{})

	for _, author := range allAuthors {
		if isIgnored(author, conf) || isBot(author) {
			continue
		}

		author, err = backend.PopulateAuthor(author, conf)
		if err != nil {
			return "", err
		}

		// check again when we have more fields
		if isIgnored(author, conf) || isBot(author) {
			continue
		}

		var uniqKeys, extraKeys, allKeys []string

		if author.Email != "" {
			uniqKeys = append(uniqKeys, author.Email)
		}
		if author.Profile != "" {
			uniqKeys = append(uniqKeys, author.Profile)
		}
		if author.Login != "" {
			extraKeys = append(uniqKeys, author.Login)
		}
		if spaceRx.MatchString(author.Name) {
			uniqKeys = append(uniqKeys, author.Name)
		} else {
			extraKeys = append(extraKeys, author.Name)
		}

		allKeys = make([]string, 0)
		allKeys = append(allKeys, uniqKeys...)
		allKeys = append(allKeys, extraKeys...)

		found := false
		for _, key := range allKeys {
			_, found = seenAuthors[key]
			if found {
				break
			}
		}

		if conf.Append {
			for n, keys := range [][]string{uniqKeys, extraKeys} {
				var nMatches, minMatches int
				if n == 0 {
					// at least one key from uniqKeys
					minMatches = 1
				} else {
					// or at least two keys from extraKeys
					minMatches = 2
				}

				for _, key := range keys {
					if match.ContainsAlike(content, key) {
						nMatches++
					}
				}

				if nMatches >= minMatches {
					found = true
					break
				}
			}
		}

		if !found {
			added += 1
			index += 1
			author.Index = index

			line, err := formatAuthor(author, conf)
			if err != nil {
				return "", err
			}

			if conf.Pipe {
				// In --pipe mode, print immediately.
				fmt.Print(line)
			} else {
				logs.Infof("new: %s <%s> %s",
					author.Name, author.Email, author.Login)
				content += line
			}
		} else {
			logs.Debugf("dup: %s <%s> %s",
				author.Name, author.Email, author.Login)
		}

		for _, key := range uniqKeys {
			seenAuthors[key] = struct{}{}
		}
	}

	if !conf.Pipe {
		if added == 0 {
			logs.Infof("no new authors")
		} else {
			logs.Infof("added %d author(s)", added)
		}
	}

	return content, nil
}

func sortKey(a defs.Author) string {
	switch {
	case a.Name != "":
		return a.Name
	case a.Email != "":
		return a.Email
	default:
		return a.Date
	}
}

func isIgnored(author defs.Author, conf defs.Config) bool {
	for _, ign := range conf.Ignore {
		ign = strings.TrimSpace(ign)
		if ign == "" {
			continue
		}

		if strings.EqualFold(author.Email, ign) || strings.EqualFold(author.Name, ign) ||
			strings.EqualFold(author.Login, ign) {
			return true
		}
	}

	return false
}

func isBot(author defs.Author) bool {
	switch author.Email {
	case "badger@gitter.im", "rocstreaming@enise.org":
		return true
	}

	switch author.Name {
	case "dependabot[bot]":
		return true
	}

	switch author.Login {
	case "gitter-badger", "rocstreaming-bot":
		return true
	}

	if strings.HasSuffix(author.Login, "[bot]") {
		return true
	}

	return false
}
