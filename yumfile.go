package main

type Yumfile struct {
	YumRepos []RepoMirror `json:"repoMirrors"`
}

type RepoMirror struct {
	BaseURL         string `json:"baseUrl,omitempty"`
	CachePath       string `json:"cachePath,omitempty"`
	EnablePlugins   bool   `json:"enablePlugins,omitempty"`
	GPGCAKey        string `json:"gpgCaKey,omitempty"`
	GPGCheck        bool   `json:"gpgCheck,omitempty"`
	GPGKeyPath      string `json:"gpgKeyPath,omitempty"`
	IncludeSources  bool   `json:"includeSources,omitempty"`
	LocalPath       string `json:"localPath,omitempty"`
	MirrorListURL   string `json:"mirrorListUrl,omitempty"`
	NewOnly         bool   `json:"newOnly,omitempty"`
	RepoDescription string `json:"repoDescription,omitempty"`
	RepoName        string `json:"repoName,omitempty"`
	RepoFile        string `json:"repoFile,omitempty"`
}
