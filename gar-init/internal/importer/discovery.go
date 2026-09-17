package importer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tvoydom/gar-init/internal/model"
)

var (
	ErrNoSupportedFiles   = errors.New("no supported GAR XML files found")
	ErrIncompleteSnapshot = errors.New("incomplete GAR XML snapshot")
	ErrDuplicateFamily    = errors.New("multiple GAR XML files for one family")
)

type SourceFile struct {
	Name   string
	Path   string
	Size   int64
	Family model.Family
}

func Discover(dir string) ([]SourceFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoSupportedFiles, dir)
		}
		return nil, fmt.Errorf("read GAR directory %s: %w", dir, err)
	}
	byFamily := make(map[string]SourceFile)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		family, ok := model.MatchFile(entry.Name())
		if !ok {
			continue
		}
		if previous, exists := byFamily[family.Key]; exists {
			return nil, fmt.Errorf("%w: %s and %s", ErrDuplicateFamily, previous.Name, entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", entry.Name(), err)
		}
		byFamily[family.Key] = SourceFile{Name: entry.Name(), Path: filepath.Join(dir, entry.Name()), Size: info.Size(), Family: family}
	}
	if len(byFamily) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoSupportedFiles, dir)
	}
	missing := make([]string, 0)
	for _, family := range model.Families() {
		if _, ok := byFamily[family.Key]; !ok {
			missing = append(missing, family.Prefix+"*.XML")
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: missing %s", ErrIncompleteSnapshot, strings.Join(missing, ", "))
	}
	files := make([]SourceFile, 0, len(byFamily))
	for _, file := range byFamily {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Family.Order < files[j].Family.Order })
	return files, nil
}
