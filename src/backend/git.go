package backend

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gavv/md-authors/src/defs"
	"github.com/gavv/md-authors/src/logs"
)

func gitCollect(conf defs.Config) ([]defs.Author, error) {
	cmdArgs := []string{"git", "log", "--format=%as;%aN;%aE", "--reverse"}

	logs.Debugf("running: %s", strings.Join(cmdArgs, " "))

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git: %w", err)
	}

	authors := []defs.Author{}
	seen := make(map[string]struct{})

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ";")
		author := defs.Author{
			Name:  split[1],
			Email: split[2],
			Date:  split[0],
		}
		if _, ok := seen[author.Name]; ok {
			continue
		}
		if _, ok := seen[author.Email]; ok {
			continue
		}
		seen[author.Name] = struct{}{}
		seen[author.Email] = struct{}{}
		authors = append(authors, author)
	}

	logs.Debugf("found %d authors in git log", len(authors))

	return authors, nil
}
