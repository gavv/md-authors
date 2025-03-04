package logs

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var (
	EnableDebug  = false
	EnableColors = haveColors()
)

func haveColors() bool {
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return len(os.Getenv("NO_COLOR")) == 0 &&
			!strings.HasPrefix(os.Getenv("TERM"), "dumb")
	}
	return false
}

func rawFprintf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format, args...)
}

// Userf prints informational message.
func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "md-authors: %s\n", fmt.Sprintf(format, args...))
}

// Userf prints debugging message.
func Debugf(format string, args ...any) {
	if EnableDebug {
		fn := rawFprintf
		if EnableColors {
			fn = color.New(color.FgHiBlack).FprintfFunc()
		}
		fn(os.Stderr, "md-authors: %s\n", fmt.Sprintf(format, args...))
	}
}

// Userf prints error message and terminates program.
func Fatalf(format string, args ...any) {
	fn := rawFprintf
	if EnableColors {
		fn = color.New(color.FgHiRed).FprintfFunc()
	}
	fn(os.Stderr, "md-authors: %s\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}
