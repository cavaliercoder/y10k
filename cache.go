package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type Cache struct {
	Path string
}

func NewCache(path string) (*Cache, error) {
	cache := &Cache{
		Path: path,
	}

	// create cache folder
	if err := cache.mkdir(path); err != nil {
		return nil, err
	}

	return cache, nil
}

func (c *Cache) NewRepoCache(repo *Repo) (*RepoCache, error) {
	cachedir := filepath.Join(c.Path, repo.ID)

	// create cache directory tree
	if err := c.mkdir(cachedir); err != nil {
		return nil, err
	}

	if err := c.mkdir(filepath.Join(cachedir, "gen")); err != nil {
		return nil, err
	}

	return &RepoCache{
		Repo: repo,
		Path: cachedir,
	}, nil
}

// mkdir creates directories required for caching, with all missing parent
// directories.
func (c *Cache) mkdir(path string) error {
	if err := os.MkdirAll(path, 0750); err != nil && os.IsNotExist(err) {
		return fmt.Errorf("Error creating cache directory %s: %v", path, err)
	}

	return nil
}

func (c *Cache) cachedir(repo *Repo) string {
	return filepath.Join(c.Path, repo.ID)
}
