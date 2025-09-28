package core

type Project struct {
	Path        string
	PubspecPath string
}

type RepoCandidate struct {
	Owner   string
	Name    string
	URL     string
	Privacy string
	Desc    string
}

type PkgSpec struct {
	Name   string
	URL    string
	Ref    string
	Subdir string
}

type ActionResult struct {
	OK      bool
	Message string
	Err     string
	Logs    []string
}

type Reco struct {
	Message   string
	Severity  string
	Rationale string
}
