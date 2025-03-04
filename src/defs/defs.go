package defs

type Config struct {
	Format string
	Sort   string

	Vcs     string
	Forge   string
	Project string

	Append bool
	Pipe   bool

	Ignore []string
}

type Author struct {
	Index int

	Date  string
	Name  string
	Email string

	Login   string
	Profile string
}
