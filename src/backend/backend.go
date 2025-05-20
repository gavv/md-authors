package backend

import (
	"github.com/gavv/md-authors/src/defs"
)

// Collect list of authors from VCS.
func CollectAuthors(conf defs.Config) ([]defs.Author, error) {
	// for now, only git is supported
	return gitCollect(conf)
}

// Populate extra author fields from forge.
func PopulateAuthor(author defs.Author, conf defs.Config) (defs.Author, error) {
	if conf.NoProject {
		return author, nil
	}

	// for now, only github is supported
	return githubPopulate(author, conf)
}
