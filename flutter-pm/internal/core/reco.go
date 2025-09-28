package core

// SuggestPopularPkgs returns a small static set of suggested packages as placeholders.
func SuggestPopularPkgs() []Reco {
	return []Reco{
		{Message: "Consider state management with riverpod", Severity: "info", Rationale: "popular, testable"},
		{Message: "Add dio for HTTP", Severity: "info", Rationale: "flexible networking"},
		{Message: "Try go_router for navigation", Severity: "info", Rationale: "simple declarative routing"},
	}
}
