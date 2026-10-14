package events

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gar_converter/internal/config"
	"gar_converter/internal/download"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDrainLiveRabbitMQ(t *testing.T) {
	url := os.Getenv("GAR_TEST_RABBITMQ_URL")
	if url == "" {
		t.Skip("set GAR_TEST_RABBITMQ_URL for live broker test")
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	queue := fmt.Sprintf("fias.parser.test.%d", time.Now().UnixNano())
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	defer ch.QueueDelete(queue, false, false, false)
	message := []byte(`{"event":"download.completed","version_id":20261013,"kind":"delta","file_path":"/app/data/downloads/20261013_delta.zip"}`)
	if err := ch.PublishWithContext(context.Background(), "", queue, false, false, amqp.Publishing{Body: message}); err != nil {
		t.Fatal(err)
	}
	called := 0
	err = Drain(context.Background(), config.RabbitMQConfig{URL: url, Queue: queue}, func(_ context.Context, kind string, result download.ExtractResult) error {
		called++
		if kind != "delta" || result.VersionID != 20261013 || result.ArchivePath != "/app/data/downloads/20261013_delta.zip" {
			t.Fatalf("unexpected event: %s %+v", kind, result)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("called %d times, want 1", called)
	}
	_, ok, err := ch.Get(queue, true)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("message was not acknowledged")
	}
}
