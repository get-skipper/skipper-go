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
