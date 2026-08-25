package download

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	config "gar_converter/internal/config"
)

// LastInfo is the response of the FIAS GetLastDownloadFileInfo web service.
// The service returns JSON (and may historically have returned XML).
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
	cfg    config.FiasConfig
	client *http.Client
}

func New(cfg config.FiasConfig) *Downloader {
	return &Downloader{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Minute},
	}
}

// LastInfo fetches info about the latest GAR version from last_info_url.
func (d *Downloader) LastInfo() (*LastInfo, error) {
	if d.cfg.LastInfoURL == "" {
		return nil, fmt.Errorf("fias.last_info_url is empty")
	}

	resp, err := d.client.Get(d.cfg.LastInfoURL)
	if err != nil {
		return nil, fmt.Errorf("get last info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get last info: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read last info: %w", err)
	}

	info := &LastInfo{}
	if err := json.Unmarshal(body, info); err != nil {
		if xerr := xml.Unmarshal(body, info); xerr != nil {
			return nil, fmt.Errorf("decode last info: json: %v; xml: %v", err, xerr)
		}
	}
	return info, nil
}

// Full downloads the full GAR XML archive and extracts it into dest.
func (d *Downloader) Full(dest string) error {
	return d.downloadAndExtract(d.archiveURL(true), d.cfg.FullZipName(), dest)
}

// Delta downloads the GAR delta XML archive and extracts it into dest.
func (d *Downloader) Delta(dest string) error {
	return d.downloadAndExtract(d.archiveURL(false), d.cfg.DeltaZipName(), dest)
}

// archiveURL returns the archive URL from the last download info (preferring
// FiasCompleteXmlUrl/FiasDeltaXmlUrl), falling back to fias.url + archive
// name when the info service is unavailable.
func (d *Downloader) archiveURL(full bool) string {
	if info, err := d.LastInfo(); err == nil {
		if full {
			if info.FiasCompleteXmlUrl != "" {
				return info.FiasCompleteXmlUrl
			}
			if info.GarXMLFullURL != "" {
				return info.GarXMLFullURL
			}
		} else {
			if info.FiasDeltaXmlUrl != "" {
				return info.FiasDeltaXmlUrl
			}
			if info.GarXMLDeltaURL != "" {
				return info.GarXMLDeltaURL
			}
		}
	}
	return ""
}

func (d *Downloader) downloadAndExtract(infoURL, fallbackName, dest string) error {
	zipURL := infoURL
	if zipURL == "" {
		if d.cfg.URL == "" {
			return fmt.Errorf("fias.url is empty and last info has no archive URL")
		}
		zipURL = strings.TrimRight(d.cfg.URL, "/") + "/" + fallbackName
	}

	fmt.Printf("Downloading %s\n", zipURL)

	tmp, err := os.CreateTemp("", "gar_*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	resp, err := d.client.Get(zipURL)
	if err != nil {
		tmp.Close()
		return fmt.Errorf("download %s: %w", zipURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		return fmt.Errorf("download %s: unexpected status %s", zipURL, resp.Status)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("save %s: %w", zipURL, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	zr, err := zip.OpenReader(tmpName)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", zipURL, err)
	}
	defer zr.Close()

	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("clean %s: %w", dest, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}

	if err := extractZip(&zr.Reader, dest); err != nil {
		return fmt.Errorf("extract %s: %w", zipURL, err)
	}
	if err := flattenSingleRootDir(dest); err != nil {
		return fmt.Errorf("flatten %s: %w", dest, err)
	}

	fmt.Printf("Extracted to %s\n", dest)
	return nil
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
