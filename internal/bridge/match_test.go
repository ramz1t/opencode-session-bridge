package bridge

import (
	"testing"
	"time"

	"opencode-session-bridge/internal/opencode"
)

func TestRankPrefersPathSegmentThenRunningThenRecent(t *testing.T) {
	base := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	sessions := []opencode.Session{
		{ID: "old-exact", Directory: "/home/dev/src/my-app", Updated: base},
		{ID: "new-exact", Directory: "/home/dev/src/my-app", Updated: base.Add(time.Hour)},
		{ID: "substring", Directory: "/home/dev/src/my-app-old", Updated: base.Add(2 * time.Hour)},
		{ID: "other", Directory: "/home/dev/src/other-app", Updated: base.Add(3 * time.Hour)},
	}

	got := Rank(sessions, map[string]bool{}, Filter{Project: "my-app"})
	assertOrder(t, got, "new-exact", "old-exact", "substring")

	got = Rank(sessions, map[string]bool{"old-exact": true}, Filter{Project: "my-app"})
	assertOrder(t, got, "old-exact", "new-exact", "substring")
}

func TestRankExcludesCallerSession(t *testing.T) {
	sessions := []opencode.Session{
		{ID: "ses_self", Directory: "/work/app", Updated: time.Now()},
		{ID: "ses_other", Directory: "/work/app", Updated: time.Now().Add(-time.Minute)},
	}

	got := Rank(sessions, nil, Filter{Project: "app", Exclude: "ses_self"})
	assertOrder(t, got, "ses_other")
}

func TestRankActiveOnlyAndTitle(t *testing.T) {
	sessions := []opencode.Session{
		{ID: "ses_1", Title: "Image version bump", Directory: "/work/app", Updated: time.Now()},
		{ID: "ses_2", Title: "Refactor parser", Directory: "/work/app", Updated: time.Now()},
	}

	got := Rank(sessions, map[string]bool{"ses_2": true}, Filter{ActiveOnly: true})
	assertOrder(t, got, "ses_2")

	got = Rank(sessions, nil, Filter{Title: "IMAGE"})
	assertOrder(t, got, "ses_1")
}

func TestRankNoMatch(t *testing.T) {
	sessions := []opencode.Session{{ID: "ses_1", Directory: "/work/other-app"}}
	if got := Rank(sessions, nil, Filter{Project: "my-app"}); len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}

func assertOrder(t *testing.T, got []opencode.Session, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d sessions, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("position %d: got %s, want %s", i, got[i].ID, id)
		}
	}
}
