package gen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/gavv/md-authors/src/defs"
	"github.com/gavv/md-authors/src/logs"
)

// Update author blocks in markdown file.
// If --append is set, only appends new authors to the block and
// doesn't touch original block contents.
func ProcessFile(path string, conf defs.Config) error {
	logs.Debugf("processing %q", path)

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("can't open %q: %w", path, err)
	}
	defer file.Close()

	var oldContent, newContent bytes.Buffer

	scanner := bufio.NewScanner(file)

	var (
		blockBuilder strings.Builder
		blockFlag    bool
		lineNo       int
	)

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		oldContent.WriteString(line)
		oldContent.WriteString("\n")

		// begin block
		if matched, _ := regexp.MatchString(
			`^\s*<!--\s*authors\s*-->\s*$`, line); matched {
			if blockFlag {
				return fmt.Errorf(
					"can't process %q: unpaired <!--authors-->/<!--endauthors--> at line %d",
					path, lineNo)
			}

			newContent.WriteString(line)
			newContent.WriteString("\n")

			blockFlag = true
			continue
		}

		// end block
		if matched, _ := regexp.MatchString(
			`^\s*<!--\s*endauthors\s*-->\s*$`, line); matched {
			if !blockFlag {
				return fmt.Errorf(
					"can't process %q: unpaired <!--authors-->/<!--endauthors--> at line %d",
					path, lineNo)
			}

			content := blockBuilder.String()
			content, err = regenerateBlock(content, conf)
			if err != nil {
				return fmt.Errorf("can't open %q: %w", path, err)
			}

			newContent.WriteString(content)
			newContent.WriteString(line)
			newContent.WriteString("\n")

			blockFlag = false
			blockBuilder.Reset()
			continue
		}

		// inside block
		if blockFlag {
			blockBuilder.WriteString(line)
			blockBuilder.WriteString("\n")
			continue
		}

		// outside of block
		newContent.WriteString(line)
		newContent.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("can't read %q: %w", path, err)
	}

	if blockFlag {
		return fmt.Errorf(
			"can't process %q: unpaired <!--authors-->/<!--endauthors--> at line %d",
			path, lineNo)
	}

	if !bytes.Equal(newContent.Bytes(), oldContent.Bytes()) {
		_, err = file.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("can't write %q: %w", path, err)
		}

		err = file.Truncate(0)
		if err != nil {
			return fmt.Errorf("can't write %q: %w", path, err)
		}

		_, err = file.Write(newContent.Bytes())
		if err != nil {
			return fmt.Errorf("can't write %q: %w", path, err)
		}
	}

	return nil
}

// Print author block to stdout.
// If --append is set, first read current content from stdin,
// and then print only new authors.
func ProcessPipe(conf defs.Config) error {
	content := ""

	if conf.Append {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("can't read stdin: %w", err)
		}
		content = string(b)
	}

	_, err := generateAuthors(content, conf)
	return err
}
