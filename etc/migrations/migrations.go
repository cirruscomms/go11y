package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"regexp"
)

//go:embed *.sql
var Filesystem embed.FS

var MigPattern = regexp.MustCompile(`^[0-9]{4}_.+\.sql$`)

type Collection struct {
	Filesystem embed.FS
	Migrations []fs.DirEntry
}

func New() (collection Collection, fault error) {
	files, _ := Filesystem.ReadDir(".")

	migrations := []fs.DirEntry{}
	for _, f := range files {
		if !f.IsDir() && MigPattern.MatchString(f.Name()) {
			migrations = append(migrations, f)
		}
	}

	return Collection{
		Filesystem: Filesystem,
		Migrations: migrations,
	}, nil
}

func (c Collection) Steps() (number int32) {
	return int32(len(c.Migrations))
}

func (c Collection) ReadDir(name string) ([]fs.FileInfo, error) {
	files, err := c.Filesystem.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("could not get the files from the embedded filesystem: %w", err)
	}

	var r []os.FileInfo

	for _, f := range files {
		fi, _ := f.Info()

		if !MigPattern.MatchString(fi.Name()) {
			continue
		}

		r = append(r, fi)
	}

	return r, nil
}

func (c Collection) ReadFile(name string) (contents []byte, fault error) {
	b, err := c.Filesystem.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s' from embedded filesystem: %w", name, err)
	}

	return b, nil
}

func (c Collection) Glob(pattern string) (matches []string, fault error) {
	matches, err := fs.Glob(c.Filesystem, pattern)
	if err != nil {
		return nil, fmt.Errorf("could not get glob matches for pattern '%s': %w", pattern, err)
	}

	filteredMatches := []string{}
	for _, m := range matches {
		if !MigPattern.MatchString(m) {
			continue
		}
		filteredMatches = append(filteredMatches, m)
	}

	return filteredMatches, nil
}

func (c Collection) Open(name string) (file fs.File, fault error) {
	f, err := c.Filesystem.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file '%s' from embedded filesystem: %w", name, err)
	}

	return f, nil
}
