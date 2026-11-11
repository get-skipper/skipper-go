package core

import "testing"

// ---- SKIPPER_SYNC_ALLOW_DELETE -----------------------------------------------

func TestSyncAllowDelete_DefaultIsFalse(t *testing.T) {
	t.Setenv("SKIPPER_SYNC_ALLOW_DELETE", "")
	if SyncAllowDelete() {
		t.Error("expected SyncAllowDelete() == false when env var is unset")
	}
}

func TestSyncAllowDelete_ExplicitTrue(t *testing.T) {
	t.Setenv("SKIPPER_SYNC_ALLOW_DELETE", "true")
	if !SyncAllowDelete() {
		t.Error("expected SyncAllowDelete() == true")
	}
}

func TestSyncAllowDelete_CaseInsensitive(t *testing.T) {
	t.Setenv("SKIPPER_SYNC_ALLOW_DELETE", "TRUE")
	if !SyncAllowDelete() {
		t.Error("expected SyncAllowDelete() == true for uppercase value")
	}
}

func TestSyncAllowDelete_ExplicitFalse(t *testing.T) {
	t.Setenv("SKIPPER_SYNC_ALLOW_DELETE", "false")
	if SyncAllowDelete() {
		t.Error("expected SyncAllowDelete() == false")
	}
}

func TestSyncAllowDelete_ArbitraryStringIsFalse(t *testing.T) {
	t.Setenv("SKIPPER_SYNC_ALLOW_DELETE", "yes")
	if SyncAllowDelete() {
		t.Error("expected SyncAllowDelete() == false for non-'true' string")
	}
}

// ---- staleRows ----------------------------------------------------------------

func TestStaleRows_UsesRealRowIndexWithBlankRows(t *testing.T) {
	// Sheet: 0 header, 1 "a.go > keep", 2 blank (skipped on fetch), 3 "a.go > gone".
	entries := []TestEntry{
		{TestID: "a.go > keep", Row: 1},
		{TestID: "a.go > gone", Row: 3},
	}
	set := func(keys ...string) map[string]struct{} {
		m := map[string]struct{}{}
		for _, k := range keys {
			m[k] = struct{}{}
		}
		return m
	}
	got := staleRows(entries, set("a.go"), set("a.go"), set("a.go > keep"))
	if len(got) != 1 || got[0] != 3 {
		t.Fatalf("expected [3], got %v", got)
	}
}
