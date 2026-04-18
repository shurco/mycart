package update

import "testing"

func TestRelease_FindAssetBySuffix(t *testing.T) {
	t.Parallel()

	r := &release{
		Assets: []*ReleaseAsset{
			{Name: "app_linux-amd64.tar.gz"},
			{Name: "app_darwin-arm64.tar.gz"},
			{Name: "app_windows-amd64.zip"},
		},
	}

	asset, err := r.findAssetBySuffix("_darwin-arm64.tar.gz")
	if err != nil {
		t.Fatalf("findAssetBySuffix: %v", err)
	}
	if asset.Name != "app_darwin-arm64.tar.gz" {
		t.Errorf("wrong asset: %s", asset.Name)
	}

	if _, err := r.findAssetBySuffix("_freebsd-amd64.tar.gz"); err == nil {
		t.Error("expected error for missing suffix")
	}
	if _, err := r.findAssetBySuffix(""); err == nil {
		t.Error("empty suffix must return an error")
	}
}

func TestRelease_Getters(t *testing.T) {
	t.Parallel()

	r := &release{Name: "v1.2.3", Url: "https://example/v1.2.3"}
	if r.GetName() != "v1.2.3" {
		t.Errorf("GetName: %q", r.GetName())
	}
	if r.GetUrl() != "https://example/v1.2.3" {
		t.Errorf("GetUrl: %q", r.GetUrl())
	}
}
