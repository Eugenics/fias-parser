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
		versionID, err := downloader.Full(XmlFilesFullPath)
		if err != nil {
			panic(err)
		}
		recordVersion(cfg, "extracted", "full", versionID, "", "")

	case "3":
		fmt.Println("Распаковка дельты ГАР")
		versionID, err := downloader.Delta(XmlFilesDeltaPath)
		if err != nil {
			panic(err)
		}
		recordVersion(cfg, "extracted", "delta", versionID, "", "")

	default:
		fmt.Println(usage)
		return
	}

	timeEnd := time.Now()
	fmt.Printf("Execution time: %s\n", timeEnd.Sub(timeStart))
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
	for _, layout := range []string{"2006.01.02", "2006-01-02", "02.01.2006"} {
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if fileVersionID == 0 {
		fmt.Println("Не удалось определить локальную версию, запись версии пропущена")
		return
	}
	info := &download.LastInfo{VersionID: fileVersionID, TextVersion: textVersion, Date: dateText}

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
