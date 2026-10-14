package events

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"gar_converter/internal/config"
	"gar_converter/internal/download"

	amqp "github.com/rabbitmq/amqp091-go"
)

type DownloadCompleted struct {
	Event     string `json:"event"`
	VersionID int64  `json:"version_id"`
	Kind      string `json:"kind"`
	FilePath  string `json:"file_path"`
}

// Drain processes queued downloads once. A failed delivery remains queued for retry.
func Drain(ctx context.Context, cfg config.RabbitMQConfig, process func(context.Context, string, download.ExtractResult) error) error {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return fmt.Errorf("connect RabbitMQ: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	defer ch.Close()
	if _, err := ch.QueueDeclare(cfg.Queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		message, ok, err := ch.Get(cfg.Queue, false)
		if err != nil {
			return fmt.Errorf("read queue: %w", err)
		}
		if !ok {
			return nil
		}
		var event DownloadCompleted
		err = json.Unmarshal(message.Body, &event)
		if err == nil {
			if event.Event != "download.completed" || (event.Kind != "full" && event.Kind != "delta") || event.VersionID <= 0 || event.FilePath == "" || filepath.Base(event.FilePath) != fmt.Sprintf("%d_%s.zip", event.VersionID, event.Kind) {
				err = fmt.Errorf("invalid download.completed event")
			}
		}
		if err == nil {
			err = process(ctx, event.Kind, download.ExtractResult{VersionID: event.VersionID, ArchivePath: event.FilePath})
		}
		if err != nil {
			if nackErr := message.Nack(false, true); nackErr != nil {
				return fmt.Errorf("%v; nack: %w", err, nackErr)
			}
			return fmt.Errorf("process queued download: %w", err)
		}
		if err := message.Ack(false); err != nil {
			return fmt.Errorf("ack queued download: %w", err)
		}
	}
}
