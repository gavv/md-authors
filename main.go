package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	"github.com/gavv/md-authors/src/cache"
	"github.com/gavv/md-authors/src/defs"
	"github.com/gavv/md-authors/src/gen"
	"github.com/gavv/md-authors/src/logs"
)

var builtinFormats = map[string]string{
	// Example:
	//  1. Ford Prefect `@Ix`
	"modern": "{index}. {name} `{login?}`\\n",

	// Example:
	//  - Ford Prefect `Ix` (<ford@betelgeuse7.sid>)
	"classic": "- {name} `{login?}` (<{email|profile?}>)\\n",
}

func main() {
	var conf defs.Config

	fset := pflag.NewFlagSet("md-authors", pflag.ContinueOnError)

	fset.SortFlags = false
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [FILES]...\n\n", fset.Name())
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		fset.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
When one or more FILES specified, the tool finds <!-- authors --> block
in each file and replaces its contents with the new up-to-date list.
When --pipe is specified, the tool instead writes authors list to stdout.

If --append is specified, the old contents is kept unaffected, and only
new authors missing in old contents are appended to the end. In case of
--pipe, old contents is read from stdin.

Author list is formatted according to FORMAT SPEC (--format option).
It is a string that can contain literal characters, escape sequences
like \n or \{, and FORMAT FIELDS.

For example, this format spec:
 * {name} ({email?})\n

Would produce lines like this:
 * Ford Prefect (ford@betelgeuse7.sid)

FIELDS SYNTAX:
  {foo}         replaced with value of field 'foo'
  {foo?}        same, but if 'foo' is empty, removes adjustent non-
                whitespace characters
  {foo|bar}     same, but if 'foo' is empty, uses 'bar'
  {foo|bar?}    same, but if 'foo' and 'bar' are both empty, removes
                adjustent non-whitespace characters

FIELDS LIST:
  index         entry number
  date          date of first contribution
  name          full name
  email         email address
  login         github login
  profile       github profile url

FORMAT SPEC can be also a NAME of predefined spec:
`)
		var specs []string
		for k := range builtinFormats {
			specs = append(specs, k)
		}
		sort.Strings(specs)
		for _, k := range specs {
			fmt.Fprintf(os.Stderr, "  %s\n      \"%s\"\n", k, builtinFormats[k])
		}
		fmt.Fprintf(os.Stderr, `
Supported SORT orders (for --sort option):
  date          by first contribution, oldest first
  name          by name, alphabetically

Supported VCS backends (for --vcs option):
  git

Supported FORGE backends (for --forge option):
  none
  github

VCS is used to collect list of authors, and FORGE is used to
collect additional or missing fields.

PROJECT (for --project option) defines project name. For github
it has form "user/repo". By default it is auto-detected.

EXAMPLES:
  md-authors -f modern -a AUTHORS.md
  md-authors --pipe --format "{name} {email}"
`)
	}

	fset.StringVarP(&conf.Format, "format", "f", "modern", "format spec")
	fset.StringVarP(&conf.Sort, "sort", "s", "date", "sort order: date, name")
	fset.BoolVarP(&conf.Append, "append", "a", false,
		"append to list instead of replacing")
	fset.BoolVarP(&conf.Pipe, "pipe", "P", false,
		"read from stdin (if --append) and write to stdout")
	ignore := fset.StringP("ignore", "x", "",
		"comma-separated list of emails, names, and logins to ignore")
	fset.StringVar(&conf.Vcs, "vcs", "git", "version control system")
	fset.StringVar(&conf.Forge, "forge", "github", "forge service")
	fset.StringVarP(&conf.Project, "project", "p", "", "forge project")
	fset.BoolVarP(&cache.Refresh, "refresh", "r", false, "refresh cached data")
	fset.BoolVarP(&logs.EnableDebug, "debug", "d", false, "enable debug logging")
	help := fset.BoolP("help", "h", false, "print this message and exit")

	err := fset.Parse(os.Args[1:])
	if err != nil {
		os.Exit(2)
	}
	if *help {
		fset.Usage()
		os.Exit(0)
	}

	conf.Ignore = strings.Split(*ignore, ",")

	switch conf.Sort {
	case "date", "name":
	default:
		logs.Fatalf("--sort=%s not recognized", conf.Sort)
	}

	switch conf.Vcs {
	case "git":
	default:
		logs.Fatalf("--vcs=%s not recognized", conf.Vcs)
	}

	switch conf.Forge {
	case "none":
	case "github":
	default:
		logs.Fatalf("--forge=%s not recognized", conf.Forge)
	}

	if !strings.Contains(conf.Format, "{") {
		f, ok := builtinFormats[conf.Format]
		if !ok {
			logs.Fatalf("--format=%s not recognized", conf.Format)
		}
		conf.Format = f
	}

	if conf.Pipe {
		if len(fset.Args()) > 0 {
			logs.Fatalf("can't specify --pipe and files at the same time")
		}

		if err := gen.ProcessPipe(conf); err != nil {
			logs.Fatalf("%s", err)
		}
	} else {
		if len(fset.Args()) < 1 {
			logs.Fatalf("no files specified")
		}

		for _, f := range fset.Args() {
			if err := gen.ProcessFile(f, conf); err != nil {
				logs.Fatalf("%s", err)
			}
		}
	}
}
