package download

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	config "gar_converter/internal/config"
)

// LastInfo contains GAR version metadata stored in version_info.
type LastInfo struct {
	VersionID          int64  `json:"VersionId" xml:"VersionId"`
	TextVersion        string `json:"TextVersion" xml:"TextVersion"`
	FiasCompleteXmlUrl string `json:"FiasCompleteXmlUrl" xml:"FiasCompleteXmlUrl"`
	FiasDeltaXmlUrl    string `json:"FiasDeltaXmlUrl" xml:"FiasDeltaXmlUrl"`
	GarXMLFullURL      string `json:"GarXMLFullURL" xml:"GarXMLFullURL"`
	GarXMLDeltaURL     string `json:"GarXMLDeltaURL" xml:"GarXMLDeltaURL"`
	ExpDate            string `json:"ExpDate" xml:"ExpDate"`
	Date               string `json:"Date" xml:"Date"`
}

type Downloader struct {
	cfg config.FiasConfig
}

func New(cfg config.FiasConfig) *Downloader {
	return &Downloader{cfg: cfg}
}

// Full extracts the latest full GAR XML archive produced by fias-downloader.
func (d *Downloader) Full(dest string) (int64, error) {
	return d.extractLatest("full", dest)
}

// Delta extracts the latest delta GAR XML archive produced by fias-downloader.
func (d *Downloader) Delta(dest string) (int64, error) {
	return d.extractLatest("delta", dest)
}

func (d *Downloader) extractLatest(kind, dest string) (int64, error) {
	archivePath, version, err := latestArchive(d.cfg.ArchivesDir, kind)
	if err != nil {
		return 0, err
	}

	fmt.Printf("Extracting %s\n", archivePath)
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return 0, fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer zr.Close()

	if err := os.RemoveAll(dest); err != nil {
		return 0, fmt.Errorf("clean %s: %w", dest, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, fmt.Errorf("create %s: %w", dest, err)
	}

	if err := extractZip(&zr.Reader, dest); err != nil {
		return 0, fmt.Errorf("extract %s: %w", archivePath, err)
	}
	if err := flattenSingleRootDir(dest); err != nil {
		return 0, fmt.Errorf("flatten %s: %w", dest, err)
	}

	fmt.Printf("Extracted to %s\n", dest)
	return version, nil
}

// latestArchive finds the highest-version archive named <version>_<kind>.zip.
func latestArchive(dir, kind string) (string, int64, error) {
	if dir == "" {
		return "", 0, fmt.Errorf("fias.archives_dir is empty")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0, fmt.Errorf("read archives directory %s: %w", dir, err)
	}

	suffix := "_" + kind + ".zip"
	var latestVersion int64 = -1
	var latestName string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		versionText := strings.TrimSuffix(entry.Name(), suffix)
		version, err := strconv.ParseInt(versionText, 10, 64)
		if err != nil || version <= latestVersion {
			continue
		}
		latestVersion = version
		latestName = entry.Name()
	}
	if latestName == "" {
		return "", 0, fmt.Errorf("%s archive not found in %s (expected <version>_%s.zip)", kind, dir, kind)
	}
	return filepath.Join(dir, latestName), latestVersion, nil
}

func extractZip(zr *zip.Reader, dest string) error {
	for _, f := range zr.File {
		clean := filepath.Clean(filepath.FromSlash(f.Name))
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
			return fmt.Errorf("unsafe path in archive: %s", f.Name)
		}
		target := filepath.Join(dest, clean)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

// flattenSingleRootDir moves the contents of the single top-level directory
// (e.g. gar_xml/ or gar_delta_xml/) up into dest, so region folders land
// directly in dest as the importer expects.
func flattenSingleRootDir(dest string) error {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return nil
	}

	root := filepath.Join(dest, entries[0].Name())
	inner, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	hasDir := false
	for _, e := range inner {
		if e.IsDir() {
			hasDir = true
			break
		}
	}
	if !hasDir {
		return nil
	}

	for _, e := range inner {
		if err := os.Rename(filepath.Join(root, e.Name()), filepath.Join(dest, e.Name())); err != nil {
			return err
		}
	}
	return os.Remove(root)
}
