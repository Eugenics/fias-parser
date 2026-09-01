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

type ExtractResult struct {
	VersionID   int64
	ArchivePath string
}

func New(cfg config.FiasConfig) *Downloader {
	return &Downloader{cfg: cfg}
}

// Full extracts the latest full GAR XML archive produced by fias-downloader.
func (d *Downloader) Full(dest string) (ExtractResult, error) {
	return d.extractLatest("full", dest)
}

// Delta extracts the latest delta GAR XML archive produced by fias-downloader.
func (d *Downloader) Delta(dest string) (ExtractResult, error) {
	return d.extractLatest("delta", dest)
}

func (d *Downloader) Latest(kind string) (ExtractResult, error) {
	archivePath, version, err := latestArchive(d.cfg.ArchivesDir, kind)
	if err != nil {
		return ExtractResult{}, err
	}
	return ExtractResult{VersionID: version, ArchivePath: archivePath}, nil
}

func (d *Downloader) Extract(result ExtractResult, dest string) error {
	return d.extract(result, dest)
}

func (d *Downloader) extractLatest(kind, dest string) (ExtractResult, error) {
	result, err := d.Latest(kind)
	if err != nil {
		return ExtractResult{}, err
	}
	if err := d.extract(result, dest); err != nil {
		return ExtractResult{}, err
	}
	return result, nil
}

func (d *Downloader) extract(result ExtractResult, dest string) error {
	fmt.Printf("Extracting %s\n", result.ArchivePath)
	zr, err := zip.OpenReader(result.ArchivePath)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", result.ArchivePath, err)
	}
	defer zr.Close()

	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clean %s: %w", dest, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}

	if err := extractZip(&zr.Reader, dest, d.cfg.ImportedFilePrefixes); err != nil {
		return fmt.Errorf("extract %s: %w", result.ArchivePath, err)
	}
	if err := flattenSingleRootDir(dest); err != nil {
		return fmt.Errorf("flatten %s: %w", dest, err)
	}

	fmt.Printf("Extracted to %s\n", dest)
	return nil
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

func extractZip(zr *zip.Reader, dest string, importedFilePrefixes []string) error {
	progress := newExtractionProgress(zr, importedFilePrefixes)
	for _, f := range zr.File {
		clean := filepath.Clean(filepath.FromSlash(f.Name))
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
			return fmt.Errorf("unsafe path in archive: %s", f.Name)
		}
		if f.FileInfo().IsDir() || !isImportedFile(filepath.Base(clean), importedFilePrefixes) {
			continue
		}
		target := filepath.Join(dest, clean)

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

		if _, err := io.Copy(out, io.TeeReader(rc, progress)); err != nil {
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
	progress.finish()
	return nil
}

type extractionProgress struct {
	total       uint64
	extracted   uint64
	lastPercent int
}

func newExtractionProgress(zr *zip.Reader, importedFilePrefixes []string) *extractionProgress {
	progress := &extractionProgress{lastPercent: -1}
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && isImportedFile(filepath.Base(filepath.FromSlash(f.Name)), importedFilePrefixes) {
			progress.total += f.UncompressedSize64
		}
	}
	return progress
}

func (p *extractionProgress) Write(data []byte) (int, error) {
	p.extracted += uint64(len(data))
	p.print(false)
	return len(data), nil
}

func (p *extractionProgress) finish() {
	p.print(true)
	if p.total > 0 {
		fmt.Println()
	}
}

func (p *extractionProgress) print(force bool) {
	if p.total == 0 {
		return
	}
	percent := int(p.extracted * 100 / p.total)
	if percent > 100 {
		percent = 100
	}
	if !force && percent == p.lastPercent {
		return
	}
	p.lastPercent = percent
	fmt.Printf("\rExtracting: %3d%% (%s / %s)", percent, formatBytes(p.extracted), formatBytes(p.total))
}

func formatBytes(size uint64) string {
	const (
		kilobyte = 1024
		megabyte = 1024 * kilobyte
		gigabyte = 1024 * megabyte
	)
	switch {
	case size >= gigabyte:
		return fmt.Sprintf("%.1f GiB", float64(size)/gigabyte)
	case size >= megabyte:
		return fmt.Sprintf("%.1f MiB", float64(size)/megabyte)
	case size >= kilobyte:
		return fmt.Sprintf("%.1f KiB", float64(size)/kilobyte)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func isImportedFile(name string, importedFilePrefixes []string) bool {
	if strings.EqualFold(name, "version.txt") {
		return true
	}

	upperName := strings.ToUpper(name)
	if filepath.Ext(upperName) != ".XML" {
		return false
	}
	for _, prefix := range importedFilePrefixes {
		upperPrefix := strings.ToUpper(prefix)
		if strings.HasPrefix(upperName, upperPrefix) {
			remainder := strings.TrimPrefix(upperName, upperPrefix)
			// GAR data file names put a date/version immediately after the
			// entity name. This keeps similarly named unsupported entities,
			// such as AS_ADDR_OBJ_DIVISION, out of the extraction result.
			if len(remainder) > 0 && remainder[0] >= '0' && remainder[0] <= '9' {
				return true
			}
		}
	}
	return false
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
