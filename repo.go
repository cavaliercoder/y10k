package main

type Repo struct {
	ID             string
	Parameters     map[string]string
	CachePath      string
	EnablePlugins  bool
	IncludeSources bool
	LocalPath      string
	NewOnly        bool
	DeleteRemoved  bool
	GPGCheck       bool
	Architecture   string
	YumfilePath    string
	YumfileLineNo  int
}

func NewRepo() *Repo {
	return &Repo{
		Parameters: make(map[string]string, 0),
	}
}

func (c *Repo) Validate() error {
	if c.ID == "" {
		return NewErrorf("Upstream repository has no ID specified (in %s:%d)", c.YumfilePath, c.YumfileLineNo)
	}

	if c.Parameters["mirrorlist"] == "" && c.Parameters["baseurl"] == "" {
		return NewErrorf("Upstream repository for '%s' has no mirror list or base URL (in %s:%d)", c.ID, c.YumfilePath, c.YumfileLineNo)
	}

	return nil
}
