package main

import (
	"context"
	"fmt"
	"gar_converter/internal/app"
	"gar_converter/internal/config"
	"gar_converter/internal/download"
	"gar_converter/internal/events"
	"gar_converter/internal/repository"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	XmlFilesFullPath  = "./source/xml/full"
	XmlFilesDeltaPath = "./source/xml/delta"
	versionFileName   = "version.txt"
)

const usage = "Укажите тип загрузки: 0 - полная, 1 - дельта, 2 - распаковать полный архив, 3 - распаковать архив дельты, 5 - загрузить все дельты по порядку"

func main() {
	timeStart := time.Now()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		panic(err)
	}

	downloader := download.New(cfg.Fias)
	repo, err := repository.NewPostgresRepository(ctx, cfg.Database.DSN, int32(cfg.Importer.Workers))
	if err != nil {
		panic(fmt.Sprintf("Не удалось подключиться к БД: %v", err))
	}
	defer repo.Close()

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println(usage)
		return
	}
	if cfg.RabbitMQ.Mode == "rabbitmq" && args[0] != "5" {
		fmt.Println("В режиме RabbitMQ используйте команду 5 для обработки очереди")
		return
	}

	switch args[0] {
	case "0":
		fmt.Println("Полная загрузка")
		fileVersionID, _, _, hasVersion := readVersionFile(XmlFilesFullPath)
		if hasVersion && versionImported(ctx, repo, fileVersionID, "full") {
			fmt.Printf("Версия %d уже импортирована в БД, импорт пропущен\n", fileVersionID)
			break
		}
		if !hasVersion {
			panic("файл version.txt отсутствует или содержит некорректную дату")
		}
		if err := app.Run(ctx, XmlFilesFullPath, cfg, repo); err != nil {
			panic(err)
		}
		if err := markVersionImported(ctx, repo, fileVersionID, "full"); err != nil {
			panic(err)
		}

	case "1":
		fmt.Println("Дельта загрузка")
		fileVersionID, _, _, hasVersion := readVersionFile(XmlFilesDeltaPath)
		if hasVersion && versionImported(ctx, repo, fileVersionID, "delta") {
			fmt.Printf("Версия %d уже импортирована в БД, импорт пропущен\n", fileVersionID)
			break
		}
		if !hasVersion {
			panic("файл version.txt отсутствует или содержит некорректную дату")
		}
		if err := app.Run(ctx, XmlFilesDeltaPath, cfg, repo); err != nil {
			panic(err)
		}
		if err := markVersionImported(ctx, repo, fileVersionID, "delta"); err != nil {
			panic(err)
		}

	case "2":
		fmt.Println("Распаковка полной выгрузки ГАР")
		result, err := downloader.Latest("full")
		if err != nil {
			panic(err)
		}
		if extractionBlocked(ctx, repo, result.VersionID, "full") {
			break
		}
		if err := downloader.Extract(result, XmlFilesFullPath); err != nil {
			panic(err)
		}
		if err := recordExtractedVersion(ctx, repo, cfg, "full", XmlFilesFullPath, result); err != nil {
			panic(err)
		}

	case "3":
		fmt.Println("Распаковка дельты ГАР")
		result, err := downloader.Latest("delta")
		if err != nil {
			panic(err)
		}
		if extractionBlocked(ctx, repo, result.VersionID, "delta") {
			break
		}
		if err := downloader.Extract(result, XmlFilesDeltaPath); err != nil {
			panic(err)
		}
		if err := recordExtractedVersion(ctx, repo, cfg, "delta", XmlFilesDeltaPath, result); err != nil {
			panic(err)
		}

	case "5":
		fmt.Println("Последовательная загрузка дельт ГАР")
		process := func(ctx context.Context, kind string, result download.ExtractResult) error {
			imported, err := repo.IsVersionImported(ctx, result.VersionID, kind)
			if err != nil {
				return err
			}
			if imported {
				fmt.Printf("Версия %d (%s) уже импортирована, пропущена\n", result.VersionID, kind)
				return nil
			}
			dest := XmlFilesDeltaPath
			if kind == "full" {
				dest = XmlFilesFullPath
			}
			if err := downloader.Extract(result, dest); err != nil {
				return err
			}
			if err := recordExtractedVersion(ctx, repo, cfg, kind, dest, result); err != nil {
				return err
			}
			if err := app.Run(ctx, dest, cfg, repo); err != nil {
				return err
			}
			return markVersionImported(ctx, repo, result.VersionID, kind)
		}
		if cfg.RabbitMQ.Mode == "rabbitmq" {
			err := events.Drain(ctx, cfg.RabbitMQ, func(ctx context.Context, kind string, result download.ExtractResult) error {
				result.SourceArchivePath = result.ArchivePath
				result.ArchivePath = filepath.Join(cfg.Fias.ArchivesDir, filepath.Base(result.ArchivePath))
				return process(ctx, kind, result)
			})
			if err != nil {
				panic(err)
			}
			break
		}
		archives, err := downloader.Archives("delta")
		if err != nil {
			panic(err)
		}
		if err := processDeltaArchives(
			ctx,
			archives,
			func(ctx context.Context, versionID int64) (bool, error) {
				return repo.IsVersionImported(ctx, versionID, "delta")
			},
			func(ctx context.Context, result download.ExtractResult) error { return process(ctx, "delta", result) },
		); err != nil {
			panic(err)
		}

	default:
		fmt.Println(usage)
		return
	}

	timeEnd := time.Now()
	fmt.Printf("Execution time: %s\n", timeEnd.Sub(timeStart))
}

func processDeltaArchives(
	ctx context.Context,
	archives []download.ExtractResult,
	isImported func(context.Context, int64) (bool, error),
	process func(context.Context, download.ExtractResult) error,
) error {
	for _, archive := range archives {
		imported, err := isImported(ctx, archive.VersionID)
		if err != nil {
			return fmt.Errorf("проверить delta-версию %d: %w", archive.VersionID, err)
		}
		if imported {
			fmt.Printf("Версия %d уже импортирована в БД, пропущена\n", archive.VersionID)
			continue
		}
		if err := process(ctx, archive); err != nil {
			return fmt.Errorf("загрузить delta-версию %d: %w", archive.VersionID, err)
		}
	}
	return nil
}

func extractionBlocked(ctx context.Context, repo *repository.PostgresRepository, versionID int64, fileType string) bool {
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

func recordExtractedVersion(ctx context.Context, repo *repository.PostgresRepository, cfg *config.Config, fileType, dir string, result download.ExtractResult) error {
	versionID, _, dateText, ok := readVersionFile(dir)
	if !ok {
		fmt.Println("Не удалось прочитать дату из version.txt, запись версии пропущена")
		return fmt.Errorf("не удалось прочитать дату из %s", filepath.Join(dir, versionFileName))
	}

	info := &download.LastInfo{
		VersionID:   versionID,
		TextVersion: "БД ФИАС от " + dateText,
		Date:        dateText,
		ExpDate:     time.Now().Add(cfg.Fias.ExpDelta).Format(time.RFC3339),
	}
	if fileType == "full" {
		info.GarXMLFullURL = versionArchivePath(result)
	} else {
		info.GarXMLDeltaURL = versionArchivePath(result)
	}
	if err := repo.SaveVersionInfo(ctx, info, "extracted", fileType); err != nil {
		return fmt.Errorf("записать распакованную версию: %w", err)
	}
	return nil
}

func versionArchivePath(result download.ExtractResult) string {
	if result.SourceArchivePath != "" {
		return result.SourceArchivePath
	}
	return result.ArchivePath
}

// versionImported reports whether the given version has already been imported
// into the DB. DB failures are fatal: better to abort than to silently re-run
// an expensive import.
func versionImported(ctx context.Context, repo *repository.PostgresRepository, versionID int64, fileType string) bool {
	imported, err := repo.IsVersionImported(ctx, versionID, fileType)
	if err != nil {
		panic(fmt.Sprintf("Не удалось проверить версию в БД: %v", err))
	}
	return imported
}

func markVersionImported(ctx context.Context, repo *repository.PostgresRepository, fileVersionID int64, fileType string) error {
	if fileVersionID == 0 {
		return fmt.Errorf("не удалось определить локальную версию")
	}
	if err := repo.MarkVersionImported(ctx, fileVersionID, fileType); err != nil {
		return fmt.Errorf("отметить версию импортированной: %w", err)
	}
	fmt.Printf("Версия %d импортирована\n", fileVersionID)
	return nil
}
