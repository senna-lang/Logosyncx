// Package updater provides update checking and self-update functionality for the logos binary.
// It queries the GitHub Releases API, caches results locally to avoid hammering the API,
// and can atomically replace the running binary with a newer version.
package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	githubRepo     = "senna-lang/Logosyncx"
	apiBaseURL     = "https://api.github.com"
	releaseBaseURL = "https://github.com/senna-lang/Logosyncx/releases/download"

	cacheTTL    = 24 * time.Hour
	httpTimeout = 30 * time.Second
	userAgent   = "logos-cli"
)

// cacheEntry is the structure persisted to the local update-check cache file.
type cacheEntry struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// CheckWithCache checks whether a newer version of logos is available.
//
// It reads from a local cache file first. If the cache is fresh (< 24 h old),
// the result is returned immediately without any network call. If the cache is
// stale or missing, the GitHub Releases API is queried (respecting ctx's deadline).
//
// Returns the latest version string (e.g. "v0.3.0") when an update is available,
// or an empty string when the current version is already up to date.
// Errors are treated as non-fatal: callers should silently skip update hints on error.
func CheckWithCache(ctx context.Context, currentVersion string) (string, error) {
	if currentVersion == "dev" {
		return "", nil
	}

	cacheFile, err := cacheFilePath()
	if err != nil {
		return "", nil // non-fatal: proceed without cache
	}

	// Serve from cache when it is still fresh.
	if entry, err := readCache(cacheFile); err == nil {
		if time.Since(entry.CheckedAt) < cacheTTL {
			if semverGreater(entry.LatestVersion, currentVersion) {
				return entry.LatestVersion, nil
			}
			return "", nil
		}
	}

	// Cache is stale or missing — query the GitHub API.
	latest, err := FetchLatestVersion(ctx)
	if err != nil {
		// Network failure is non-fatal; suppress the hint for this invocation.
		return "", nil
	}

	// Persist result so the next invocation is served from cache.
	_ = writeCache(cacheFile, cacheEntry{
		LatestVersion: latest,
		CheckedAt:     time.Now(),
	})

	if semverGreater(latest, currentVersion) {
		return latest, nil
	}
	return "", nil
}

// FetchLatestVersion queries the GitHub Releases API and returns the tag name of
// the latest release (e.g. "v0.3.0").
func FetchLatestVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBaseURL, githubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode github response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in github response")
	}
	return release.TagName, nil
}

// Apply downloads targetVersion from GitHub Releases, verifies its SHA256 checksum,
// and atomically replaces the binary at execPath with the new version.
//
// targetVersion must be a full semver tag (e.g. "v0.3.0").
// execPath is typically obtained from os.Executable().
func Apply(ctx context.Context, targetVersion, execPath string) error {
	asset := assetName()
	archiveURL := fmt.Sprintf("%s/%s/%s", releaseBaseURL, targetVersion, asset)
	checksumURL := fmt.Sprintf("%s/%s/checksums.txt", releaseBaseURL, targetVersion)

	tmpDir, err := os.MkdirTemp("", "logos-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset)
	checksumPath := filepath.Join(tmpDir, "checksums.txt")

	// Download the archive and its checksum file.
	if err := downloadFile(ctx, archiveURL, archivePath); err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}
	if err := downloadFile(ctx, checksumURL, checksumPath); err != nil {
		return fmt.Errorf("download checksums.txt: %w", err)
	}

	// Verify the archive before extracting.
	if err := verifyChecksum(archivePath, checksumPath, asset); err != nil {
		return fmt.Errorf("checksum verification: %w", err)
	}

	// Extract the logos binary from the archive.
	binaryName := "logos"
	if runtime.GOOS == "windows" {
		binaryName = "logos.exe"
	}
	extractedPath := filepath.Join(tmpDir, "extracted", binaryName)
	if err := extractBinary(archivePath, filepath.Join(tmpDir, "extracted"), binaryName); err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	// Atomic replacement: write to a sibling temp file, then rename.
	if err := replaceBinary(extractedPath, execPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// assetName returns the archive filename for the current OS and architecture,
// matching the name_template configured in .goreleaser.yaml:
//
//	logos_<os>_<arch>.tar.gz  (or .zip on Windows)
func assetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("logos_%s_%s.%s", goos, goarch, ext)
}

// downloadFile fetches url and writes the response body to dest.
func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// verifyChecksum reads checksums.txt and confirms the SHA256 of archivePath
// matches the entry for archiveName.
func verifyChecksum(archivePath, checksumPath, archiveName string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read checksums.txt: %w", err)
	}

	// Find the line that ends with the archive name.
	// GoReleaser emits lines in the form: "<hash>  <filename>"
	var expected string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		// Support both "hash  name" and "hash *name" formats.
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == archiveName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum entry found for %s in checksums.txt", archiveName)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash archive: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))

	if actual != expected {
		return fmt.Errorf("checksum mismatch:\n  expected: %s\n  actual:   %s", expected, actual)
	}
	return nil
}

// extractBinary extracts the named binary from a .tar.gz or .zip archive
// into destDir.
func extractBinary(archivePath, destDir, binaryName string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, destDir, binaryName)
	}
	return extractFromTarGz(archivePath, destDir, binaryName)
}

func extractFromTarGz(archivePath, destDir, binaryName string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		// Match the bare binary name regardless of any directory prefix in the archive.
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}

		destPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("create binary: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return fmt.Errorf("write binary: %w", err)
		}
		out.Close()
		return nil
	}
	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractFromZip(archivePath, destDir, binaryName string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}

		destPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create binary: %w", err)
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			return fmt.Errorf("write binary: %w", copyErr)
		}
		return nil
	}
	return fmt.Errorf("binary %q not found in zip archive", binaryName)
}

// replaceBinary atomically replaces the file at destPath with the file at srcPath.
// It preserves the file permissions of the original binary.
func replaceBinary(srcPath, destPath string) error {
	// Determine the permissions of the existing binary (fall back to 0755).
	mode := os.FileMode(0755)
	if info, err := os.Stat(destPath); err == nil {
		mode = info.Mode().Perm()
	}

	// Write to a sibling temp file in the same directory so that os.Rename is
	// atomic on the same filesystem (Unix guarantee).
	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, ".logos-update-*")
	if err != nil {
		// If we cannot write to the directory, the user likely needs sudo.
		return fmt.Errorf("cannot write to %s (try: sudo logos update): %w", dir, err)
	}
	tmpPath := tmp.Name()

	src, err := os.Open(srcPath)
	if err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	defer src.Close()

	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write new binary: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, mode); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename (try: sudo logos update): %w", err)
	}
	return nil
}

// ── cache helpers ──────────────────────────────────────────────────────────────

func cacheFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "logosyncx")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "update-check.json"), nil
}

func readCache(path string) (cacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}, err
	}
	return entry, nil
}

func writeCache(path string, entry cacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	// Write atomically via a temp file.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".update-cache-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), path)
}

// ── semver helpers ─────────────────────────────────────────────────────────────

// semverGreater returns true if version a is strictly greater than version b.
// Both values should be in "vMAJOR.MINOR.PATCH" format; the leading "v" is optional.
// Pre-release suffixes (e.g. "-beta.1") are stripped before comparison.
func semverGreater(a, b string) bool {
	av := parseSemver(a)
	bv := parseSemver(b)
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return true
		}
		if av[i] < bv[i] {
			return false
		}
	}
	return false // equal
}

// parseSemver converts a version string into a [3]int tuple [major, minor, patch].
func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Strip pre-release suffix (e.g. "1-beta.1" → "1").
		p = strings.SplitN(p, "-", 2)[0]
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}
