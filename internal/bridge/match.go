package bridge

import (
	"sort"
	"strings"

	"opencode-session-bridge/internal/opencode"
)

type Filter struct {
	Project    string
	Title      string
	ActiveOnly bool
	Exclude    string
}

// Rank returns the sessions that match f, best first. Path-segment matches
// beat substring matches, then running sessions, then most recently updated.
func Rank(sessions []opencode.Session, running map[string]bool, f Filter) []opencode.Session {
	type candidate struct {
		session opencode.Session
		score   int
	}

	var candidates []candidate
	for _, s := range sessions {
		if s.ID == f.Exclude {
			continue
		}
		if f.ActiveOnly && !running[s.ID] {
			continue
		}
		score := 1
		if f.Project != "" {
			score = projectScore(s.Directory, f.Project)
			if score == 0 {
				continue
			}
		}
		if f.Title != "" && !strings.Contains(strings.ToLower(s.Title), strings.ToLower(f.Title)) {
			continue
		}
		candidates = append(candidates, candidate{session: s, score: score})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if running[a.session.ID] != running[b.session.ID] {
			return running[a.session.ID]
		}
		return a.session.Updated.After(b.session.Updated)
	})

	ranked := make([]opencode.Session, 0, len(candidates))
	for _, c := range candidates {
		ranked = append(ranked, c.session)
	}
	return ranked
}

// projectScore is 2 when a directory path segment equals the project,
// 1 when the project is a substring of the path, and 0 otherwise.
func projectScore(directory, project string) int {
	dir := strings.ToLower(directory)
	want := strings.ToLower(strings.Trim(project, "/ "))
	if want == "" {
		return 1
	}
	for _, segment := range strings.Split(strings.Trim(dir, "/"), "/") {
		if segment == want {
			return 2
		}
	}
	if strings.Contains(dir, want) {
		return 1
	}
	return 0
}
