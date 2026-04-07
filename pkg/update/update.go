package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shurco/mycart/pkg/archive"
)

// Config is ...
type Config struct {
	Owner             string `json:"owner"`
	Repo              string `json:"repo"`
	CurrentVersion    string `json:"current_version"`
	ArchiveExecutable string `json:"archive_executable"`
}

// Init is ...
func Init(cfg *Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	asset, err := ReleaseInfo(ctx, cfg)
	if err != nil {
		return err
	}
	if asset == nil {
		// no update required
		fmt.Printf("You already have the latest mycart %s\n", cfg.CurrentVersion)
		return nil
	}

	// Downloading
	fmt.Printf("Downloading %s...\n", asset.Name)
	releaseDir := filepath.Join("./", ".lc_temp_to_delete")
	defer func() { _ = os.RemoveAll(releaseDir) }()

	assetArch := filepath.Join(releaseDir, asset.Name)
	if err := downloadFile(ctx, asset.DownloadUrl, assetArch); err != nil {
		return err
	}

	// Extracting
	fmt.Printf("Extracting %s...\n", asset.Name)
	extractDir := filepath.Join(releaseDir, "extracted_"+asset.Name)
	defer func() { _ = os.RemoveAll(extractDir) }()

	if runtime.GOOS == "windows" {
		if err := archive.ExtractZip(assetArch, extractDir); err != nil {
			return err
		}
	} else {
		if err := archive.ExtractTar(assetArch, extractDir); err != nil {
			return err
		}
	}

	// Replacing the executable
	fmt.Print("Replacing the executable...\n")
	oldExec, err := os.Executable()
	if err != nil {
		return err
	}
	renamedOldExec := oldExec + ".old"
	defer func() { _ = os.Remove(renamedOldExec) }()

	newExec := filepath.Join(extractDir, cfg.ArchiveExecutable)
	if _, err := os.Stat(newExec); err != nil {
		newExec = newExec + ".exe"
		if _, fallbackErr := os.Stat(newExec); fallbackErr != nil {
			return fmt.Errorf("the executable in the extracted path is missing or it is inaccessible: %v, %v", err, fallbackErr)
		}
	}

	if err := os.Rename(oldExec, renamedOldExec); err != nil {
		return err
	}

	// replace with the extracted binary
	if err := os.Rename(newExec, oldExec); err != nil {
		// best-effort rollback
		_ = os.Rename(renamedOldExec, oldExec)
		return err
	}

	fmt.Print("Update completed successfully! You can start the executable as usual.\n")
	return nil
}

// ReleaseInfo is ...
func ReleaseInfo(ctx context.Context, cfg *Config) (*ReleaseAsset, error) {
	latest, err := FetchLatestRelease(ctx, cfg.Owner, cfg.Repo)
	if err != nil {
		return nil, err
	}

	if compareVersions(strings.TrimPrefix(cfg.CurrentVersion, "v"), strings.TrimPrefix(latest.Tag, "v")) <= 0 {
		fmt.Printf("You already have the latest mycart %s\n", cfg.CurrentVersion)
		return nil, nil
	}

	suffix := archiveSuffix(runtime.GOOS, runtime.GOARCH)
	if suffix == "" {
		return nil, errors.New("unsupported platform")
	}

	asset, err := latest.findAssetBySuffix(suffix)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

// FetchLatestRelease is ...
func FetchLatestRelease(ctx context.Context, owner string, repo string) (*release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	rawBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("(%d) failed to fetch latest releases:\n%s", res.StatusCode, string(rawBody))
	}

	result := &release{}
	if err := json.Unmarshal(rawBody, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *release) GetName() string {
	return r.Name
}

func (r *release) GetUrl() string {
	return r.Url
}

func downloadFile(ctx context.Context, url string, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 400 {
		return fmt.Errorf("(%d) failed to send download file request", res.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return err
	}

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = dest.Close() }()

	if _, err := io.Copy(dest, res.Body); err != nil {
		return err
	}

	return nil
}

func compareVersions(a, b string) int {
	aSplit := strings.Split(a, ".")
	bSplit := strings.Split(b, ".")

	aTotal := len(aSplit)
	bTotal := len(bSplit)

	for i := 0; i < aTotal && i < bTotal; i++ {
		x, _ := strconv.Atoi(aSplit[i])
		y, _ := strconv.Atoi(bSplit[i])

		if x < y {
			return 1 // b is newer
		}

		if x > y {
			return -1 // a is newer
		}
	}

	if aTotal < bTotal {
		return 1 // b is newer
	}

	if aTotal > bTotal {
		return -1 // a is newer
	}

	return 0 // equal
}

func archiveSuffix(goos, goarch string) string {
	archiveMap := map[string]map[string]string{
		"linux": {
			"amd64": "_linux-amd64.tar.gz",
			"arm64": "_linux-arm64.tar.gz",
		},
		"darwin": {
			"amd64": "_darwin-amd64.tar.gz",
			"arm64": "_darwin-arm64.tar.gz",
		},
		"windows": {
			"amd64": "_windows-amd64.zip",
			"arm64": "_windows-arm64.zip",
		},
	}

	if archMap, ok := archiveMap[goos]; ok {
		if suffix, ok := archMap[goarch]; ok {
			return suffix
		}
	}

	return ""
}
