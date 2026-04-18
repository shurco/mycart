package update

import "testing"

func TestVersion_SetAndGet(t *testing.T) {
	// NOT t.Parallel — mutates package-level versionInfo.
	prev := versionInfo
	t.Cleanup(func() { versionInfo = prev })

	if VersionInfo() != prev {
		t.Fatalf("initial VersionInfo mismatch")
	}

	want := &Version{CurrentVersion: "1.2.3", GitCommit: "abc", BuildDate: "today"}
	SetVersion(want)

	got := VersionInfo()
	if got != want {
		t.Fatalf("VersionInfo returned %p, want %p", got, want)
	}
	if got.CurrentVersion != "1.2.3" || got.GitCommit != "abc" {
		t.Errorf("unexpected fields: %+v", got)
	}

	SetVersion(nil)
	if VersionInfo() != nil {
		t.Error("VersionInfo should return nil after SetVersion(nil)")
	}
}
