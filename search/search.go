package search

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/filebrowser/filebrowser/v2/rules"
)

type searchOptions struct {
	CaseSensitive bool
	Conditions    []condition
	Terms         []string
}

const maxDepth = 5

// Search searches for a query in a fs.
func Search(fs afero.Fs, scope, query string, checker rules.Checker, found func(path string, f os.FileInfo) error, depth int) error {
	// Check if the maximum depth has been reached
	if depth > maxDepth {
		return nil
	}

	search := parseSearch(query)

	scope = filepath.ToSlash(filepath.Clean(scope))
	scope = path.Join("/", scope)

	return afero.Walk(fs, scope, func(fPath string, f os.FileInfo, err error) error {
		fPath = filepath.ToSlash(filepath.Clean(fPath))
		fPath = path.Join("/", fPath)
		relativePath := strings.TrimPrefix(fPath, scope)
		relativePath = strings.TrimPrefix(relativePath, "/")

		if fPath == scope {
			return nil
		}

		if !checker.Check(fPath) {
			return nil
		}

		// Check if the file is a symbolic link
		stat, err := os.Lstat(fPath)
		if err != nil {
			return err
		}

		if stat.Mode()&os.ModeSymlink != 0 {
			// If it is a symbolic link, recursively search inside it
			return Search(fs, fPath, query, checker, found, depth+1)
		}

		if len(search.Conditions) > 0 {
			match := false

			for _, t := range search.Conditions {
				if t(fPath) {
					match = true
					break
				}
			}

			if !match {
				return nil
			}
		}

		if len(search.Terms) > 0 {
			for _, term := range search.Terms {
				_, fileName := path.Split(fPath)
				if !search.CaseSensitive {
					fileName = strings.ToLower(fileName)
					term = strings.ToLower(term)
				}
				if strings.Contains(fileName, term) {
					return found(relativePath, f)
				}
			}
			return nil
		}

		return found(relativePath, f)
	})
}
