package backend

import (
	"github.com/gavv/md-authors/src/defs"
)

// Collect list of authors from VCS.
func CollectAuthors(conf defs.Config) ([]defs.Author, error) {
	switch conf.Vcs {
	case "git":
		return gitCollect(conf)
	}

	return nil, nil
}

// Populate extra author fields from forge.
func PopulateAuthor(author defs.Author, conf defs.Config) (defs.Author, error) {
	switch conf.Forge {
	case "github":
		return githubPopulate(author, conf)
	}

	return author, nil
}
