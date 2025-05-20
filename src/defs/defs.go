package defs

type Config struct {
	Format string
	Sort   string

	Project   string
	NoProject bool

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
