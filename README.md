# md-authors [![Build](https://github.com/gavv/md-authors/workflows/build/badge.svg)](https://github.com/gavv/md-authors/actions)

<!-- toc -->

- [About](#about)
- [Installation](#installation)
  - [Install Go](#install-go)
  - [Install md-authors](#install-md-authors)
  - [Install gh](#install-gh)
- [Command-line options](#command-line-options)
- [Usage](#usage)
  - [Basic usage](#basic-usage)
  - [Append mode](#append-mode)
  - [Pipe mode](#pipe-mode)
  - [Format spec](#format-spec)
  - [Sort order](#sort-order)
  - [Git and GitHub](#git-and-github)
  - [Troubleshooting](#troubleshooting)
- [Caveats](#caveats)
- [History](#history)
- [Authors](#authors)
- [License](#license)

<!-- tocstop -->

## About

`md-authors` is a command-line tool that prints list of contributors from git and/or github to markdown file or stdout.

Features:

- Collect author list from git and optionally populate with additional information from github (like login, display name, contact email, etc.).

- Insert/update authors list in markdown file surrounded by `<!-- authors -->` / `<!-- endauthors -->` magic comments.

- Alternatively, print authors list to stdout in arbitrary format.

- Specify custom format string to define which fields to print and how.

- Append mode, when only new authors are printed. Useful when you want to edit previously generated list by hand and keep your edits intact.

## Installation

### Install Go

First, install Go >= 1.21.

On Ubuntu:

```
sudo apt install golang-go
```

On macOS:

```
brew install go
```

### Install md-authors

Then run this command, which will download, build, and install `md-authors` executable into `$GOPATH/bin` (it's `~/go/bin` if `GOPATH` environment variable is not set):

```
go install -v github.com/gavv/md-authors@latest
```

Alternatively, you can install from sources:

```
git clone https://github.com/gavv/md-authors.git
cd md-authors
go build
./md-authors --help
```

### Install gh

It is also recommended to install official [`gh` tool](https://cli.github.com/), add it to your PATH, and run `gh auth login` to authenticate on GitHub.

When available, `gh` is automatically used to make authenticated requests to GitHub, which have significantly higher rate limits compared to unauthenticated.

## Command-line options

```
Usage: md-authors [OPTIONS] [FILES]...

OPTIONS:
  -f, --format string    format spec (default "modern")
  -s, --sort string      sort order: date, name (default "date")
  -a, --append           append to list instead of replacing
  -P, --pipe             read from stdin (if --append) and write to stdout
  -x, --ignore string    comma-separated list of emails, names, and logins to ignore
      --vcs string       version control system (default "git")
      --forge string     forge service (default "github")
  -p, --project string   forge project
  -r, --refresh          refresh cached data
  -d, --debug            enable debug logging
  -h, --help             print this message and exit
```

## Usage

### Basic usage

Example `AUTHORS.md` file:

```
$ cat AUTHORS.md

# MyProject

List of MyProject authors, ordered by first contribution:

<!-- authors -->

<!-- endauthors -->
```

File name doesn't matter, but "AUTHORS" is a widely used one. All text besides `<!-- authors -->` / `<!-- endauthors -->` block is arbitrary.

Run tool:

```
$ md-authors AUTHORS.md
md-authors: new: Arthur Philip Dent <dent@yahoo.com> sandwich-maker
md-authors: new: Ford Prefect <ford@betelgeuse7.sid> Ix
md-authors: added 2 author(s)
```

Updated `AUTHORS.md` file:

```
$ cat AUTHORS.md

# MyProject

List of MyProject authors, ordered by first contribution:

<!-- authors -->

1. Arthur Philip Dent `sandwich-maker`
2. Ford Prefect `Ix`

<!-- endauthors -->
```

### Append mode

By default, contents of the authors block is replaced.

If `--append` is specified, the old contents of markdown block is kept unaffected, and only new authors missing in old contents are appended to the end of the block. It allows you to edit generated author list and keep your edits intact.

When used with `--pipe` (see below), old contents is read from stdin, and only new authors are printed to stdout.

In both cases old content of the block is used to detect new authors by matching names, emails, and github logins and profile urls in the content text.

The matching tries to be smart (a bit): if some unique field like email or profile url or two-word name is present in old content, it is considered a hit, but if only less unique fields are present like one-word name or nickname, then it requires at least two matches to consider it a hit.

Matching is case-insensitive. Unicode letters are converted to ASCII before comparison, so that letters like "é" and "e" will be treated as equal.

### Pipe mode

If `--pipe` is specified, the author list is written to stdout.

Example:

```
$ md-authors --pipe --format "{email};{login};{name}"
dent@yahoo.com;sandwich-maker;Arthur Philip Dent
ford@betelgeuse7.sid;Ix;Ford Prefect
```

This example uses `--format` option, described in the next section.

When `--pipe` is used together with `--append`, the tool reads old content from stdin and uses it to detect and print only new authors.

### Format spec

`--format` option defines output format of author entries.

It can be one of the predefined specs:

- `--format=modern` (default)

    Produces lines like:

    ```
    1. Arthur Philip Dent `sandwich-maker`
    2. Ford Prefect `Ix`
    ```

- `--format=classic`

    Produces lines like:

    ```
    - Arthur Philip Dent `sandwich-maker` (<dent@yahoo.com>)
    - Ford Prefect `Ix` (<ford@betelgeuse7.sid>)
    ```

Alternatively, `--format` can define custom spec. It should be a string that can mix literal characters, *escape sequences*, and *format fields*.

For example, `--format=modern` spec is equivalent to:

```
--format="{index}. {name} `{login?}`\n"
```

and `--format=classic` spec is equivalent to:

```
--format="- {name} `{login?}` (<{email|profile?}>)\n"
```

Here, `{...}` and `\n` have special meaning (they're format fields and and escape sequences, accordingly) and are substituted with something. Everything else is interpreted as a literal character, and is added as-is, including `-`, backtick, `()`, and `<>`.

**Escape sequence** is a backslash followed by any character, like `\n` or `\{`. It supports common special characters (`\n` for newline, `\t` for tab, etc). You can also use it to escape `{` or `\`.

**Format fields** have the following syntax:

- `{foo}` - replaced with value of field "foo"
- `{foo?}` - same, but if "foo" is empty, removes adjustent non-whitespace literal characters
- `{foo|bar}` - same, but if "foo" field is empty, uses field "bar"
- `{foo|bar?}` - same, but if "foo" and "bar" fields are both empty, removes adjustent non-whitespace literal characters

List of available fields:

| field       | description                                           |
|-------------|-------------------------------------------------------|
| `{index}`   | entry number, starts from 1 and increments each entry |
| `{date}`    | date of first contribution (`YYYY-MM-DD`)             |
| `{name}`    | full name                                             |
| `{email}`   | email address                                         |
| `{login}`   | github login                                          |
| `{profile}` | github profile url                                    |

Some fields may be empty/missing if this information is not available on GitHub or if GitHub support is disabled via `--forge=none` option.

Example of `|` syntax usage is to print email if it's available or profile link otherwise:

```
{email|profile}
```

The `?` syntax can be used to remove formatting adjustent to the field, e.g. in:

```
`{login?}`
```

when `login` field is empty, it is removed together with surrounding backticks, and in:

```
(<{email|profile?}>)
```

when both `email` and `profile` are empty, they are removed together with surrounding `<>` and `()`.

### Sort order

`--sort` option define in which order authors appear:

- `--sort=date` (default) - by first contribution date, oldest first
- `--sort=name` - by name, alphabetically

Note that in `--append` mode, sort order affects only newly added entries, so using `--append` together with `--sort=name` is probably not what you want.

### Git and GitHub

`--vcs` option is intended to specify type of version control system to collect list of authors. However, currently only `git` is supported.

`--forge` option defines which code forge to use to fetch additional information. Currently only two values are accepted: `github` (default) and `none`:

- With `--forge=github`, the tool queries GitHub API, needs Internet connection, and is slower.
- With `--forge=none`, the tool is completely offline and uses only information from local VCS.

`--project` option may be used to explicitly specify github repository name in form `<owner>/<repo>`. If not specified, it is automatically detected from `git remote -v`.

You're welcome to send pull requests with support for other backends, e.g. `--vcs=hg` or `--forge=gitlab`.

### Troubleshooting

Backend queries are cached in `~/.cache/mdauthors.json`, to make subsequent invocations fast. You can force re-fetching of queried fields using `--refresh` option. Or you can delete this file to clean the cache entirely.

Use `--debug` option to enable verbose logging to stderr. It may be handy to use it together with `--pipe` option.

## Caveats

Matching of authors is not 100% reliable. For example, if two entries have the same full name with 2+ space-separated words, they are considered as the same author. Both false positives and false negatives are possible occasionally, though they're rare in practice. With `--append` mode, it is safe to edit output by hand to fix inconsistencies.

When using `--sort=name` together with `--append`, only newly added authors are sorted, because in append mode the tool doesn't touch existing content of the block.

When using `{index}` field together with `--append` mode, it is assumed that one entry corresponds to one non-whitespace line. It will work incorrectly if that's not true (i.e. if you use custom `--format` option with multiple newlines).

If you have slow Internet, be patient. You can specify `--debug` if you're bored.

## History

Changelog file can be found here: [changelog](CHANGES.md).

## Authors

See [here](AUTHORS.md).

## License

[MIT](LICENSE)
