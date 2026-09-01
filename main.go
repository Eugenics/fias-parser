package main

import (
	"context"
	"fmt"
	"gar_converter/internal/app"
	"gar_converter/internal/config"
	"gar_converter/internal/download"
	"gar_converter/internal/repository"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	XmlFilesFullPath  = "./source/xml/full"
	XmlFilesDeltaPath = "./source/xml/delta"
	versionFileName   = "version.txt"
)

const usage = "Укажите тип загрузки: 0 - полная, 1 - дельта, 2 - распаковать полный архив, 3 - распаковать архив дельты"

func main() {
	timeStart := time.Now()

	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		panic(err)
	}

	downloader := download.New(cfg.Fias)

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println(usage)
		return
	}

	switch args[0] {
	case "0":
		fmt.Println("Полная загрузка")
		fileVersionID, textVersion, dateText, hasVersion := readVersionFile(XmlFilesFullPath)
		if hasVersion && versionImported(cfg, fileVersionID) {
			fmt.Printf("Версия %d уже импортирована в БД, импорт пропущен\n", fileVersionID)
			break
		}
		if !hasVersion {
			fmt.Println("Версия выгрузки: файл version.txt не найден, проверка пропущена")
		}
		if err := app.Run(XmlFilesFullPath); err != nil {
			panic(err)
		}
		recordVersion(cfg, "imported", "full", fileVersionID, textVersion, dateText)

	case "1":
		fmt.Println("Дельта загрузка")
		fileVersionID, textVersion, dateText, hasVersion := readVersionFile(XmlFilesDeltaPath)
		if hasVersion && versionImported(cfg, fileVersionID) {
			fmt.Printf("Версия %d уже импортирована в БД, импорт пропущен\n", fileVersionID)
			break
		}
		if !hasVersion {
			fmt.Println("Версия выгрузки: файл version.txt не найден, проверка пропущена")
		}
		if err := app.Run(XmlFilesDeltaPath); err != nil {
			panic(err)
		}
		recordVersion(cfg, "imported", "delta", fileVersionID, textVersion, dateText)

	case "2":
		fmt.Println("Распаковка полной выгрузки ГАР")
		result, err := downloader.Latest("full")
		if err != nil {
			panic(err)
		}
		if extractionBlocked(cfg, result.VersionID, "full") {
			break
		}
		if err := downloader.Extract(result, XmlFilesFullPath); err != nil {
			panic(err)
		}
		recordExtractedVersion(cfg, "full", XmlFilesFullPath, result)

	case "3":
		fmt.Println("Распаковка дельты ГАР")
		result, err := downloader.Latest("delta")
		if err != nil {
			panic(err)
		}
		if extractionBlocked(cfg, result.VersionID, "delta") {
			break
		}
		if err := downloader.Extract(result, XmlFilesDeltaPath); err != nil {
			panic(err)
		}
		recordExtractedVersion(cfg, "delta", XmlFilesDeltaPath, result)

	default:
		fmt.Println(usage)
		return
	}

	timeEnd := time.Now()
	fmt.Printf("Execution time: %s\n", timeEnd.Sub(timeStart))
}

func extractionBlocked(cfg *config.Config, versionID int64, fileType string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, err := repository.NewPostgresRepository(ctx, cfg.Database.DSN)
	if err != nil {
		panic(fmt.Sprintf("Не удалось подключиться к БД для проверки распаковки: %v", err))
	}
	defer repo.Close()

	blockingVersion, status, blocked, err := repo.ExtractionBlocker(ctx, versionID, fileType)
	if err != nil {
		panic(fmt.Sprintf("Не удалось проверить распакованные версии: %v", err))
	}
	if !blocked {
		return false
	}
	if blockingVersion == versionID {
		fmt.Printf("Версия %d уже обработана (status=%s), распаковка пропущена\n", blockingVersion, status)
	} else {
		fmt.Printf("Версия %d распакована, но ещё не импортирована; сначала импортируйте её. Распаковка версии %d пропущена\n", blockingVersion, versionID)
	}
	return true
}

// readVersionFile parses <dir>/version.txt. Line 1 is a date (e.g.
// "2026.08.04"), line 2 a human-readable version (e.g. "v.254"). It returns
// the version as version_info.version_id (date without dots, e.g. 20260804),
// plus the texts to store. ok is false when the file is absent or unparseable.
func readVersionFile(dir string) (versionID int64, textVersion, dateText string, ok bool) {
	data, err := os.ReadFile(filepath.Join(dir, versionFileName))
	if err != nil {
		return 0, "", "", false
	}

	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return 0, "", "", false
	}

	dateText = strings.TrimSpace(lines[0])
	for _, layout := range []string{"2006.01.02", "2006-01-02", "02.01.2006", "20060102"} {
		if t, err := time.Parse(layout, dateText); err == nil {
			textVersion = dateText
			if len(lines) == 2 && strings.TrimSpace(lines[1]) != "" {
				textVersion = strings.TrimSpace(lines[1])
			}
			versionID, _ = strconv.ParseInt(t.Format("20060102"), 10, 64)
			return versionID, textVersion, dateText, true
		}
	}
	return 0, "", "", false
}

func recordExtractedVersion(cfg *config.Config, fileType, dir string, result download.ExtractResult) {
	versionID, _, dateText, ok := readVersionFile(dir)
	if !ok {
		fmt.Println("Не удалось прочитать дату из version.txt, запись версии пропущена")
		return
	}

	info := &download.LastInfo{
		VersionID:   versionID,
		TextVersion: "БД ФИАС от " + dateText,
		Date:        dateText,
		ExpDate:     time.Now().Add(cfg.Fias.ExpDelta).Format(time.RFC3339),
	}
	if fileType == "full" {
		info.GarXMLFullURL = result.ArchivePath
	} else {
		info.GarXMLDeltaURL = result.ArchivePath
	}
	recordVersionInfo(cfg, "extracted", fileType, info)
}

// versionImported reports whether the given version has already been imported
// into the DB. DB failures are fatal: better to abort than to silently re-run
// an expensive import.
func versionImported(cfg *config.Config, versionID int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, err := repository.NewPostgresRepository(ctx, cfg.Database.DSN)
	if err != nil {
		panic(fmt.Sprintf("Не удалось подключиться к БД для проверки версии: %v", err))
	}
	defer repo.Close()

	imported, err := repo.IsVersionImported(ctx, versionID)
	if err != nil {
		panic(fmt.Sprintf("Не удалось проверить версию в БД: %v", err))
	}
	return imported
}

// recordVersion stores the version info into version_info. If fileVersionID
// is non-zero (import modes), the version comes from version.txt; FIAS
// fileType records the loaded file kind
// ("full" or "delta").
func recordVersion(cfg *config.Config, status, fileType string, fileVersionID int64, textVersion, dateText string) {
	if fileVersionID == 0 {
		fmt.Println("Не удалось определить локальную версию, запись версии пропущена")
		return
	}
	info := &download.LastInfo{VersionID: fileVersionID, TextVersion: textVersion, Date: dateText}
	recordVersionInfo(cfg, status, fileType, info)
}

func recordVersionInfo(cfg *config.Config, status, fileType string, info *download.LastInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, err := repository.NewPostgresRepository(ctx, cfg.Database.DSN)
	if err != nil {
		fmt.Printf("Не удалось подключиться к БД для записи версии: %v\n", err)
		return
	}
	defer repo.Close()

	if err := repo.SaveVersionInfo(ctx, info, status, fileType); err != nil {
		fmt.Printf("Не удалось записать версию в БД: %v\n", err)
		return
	}
	fmt.Printf("Версия %d (%s) записана в БД (status=%s)\n", info.VersionID, info.TextVersion, status)
}
