package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Message represents the chat message structure from Kafka
type Message struct {
	ConversationID int       `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Text           string    `json:"text"`
	CreatedAt      time.Time `json:"created_at"`
}

func main() {
	_ = godotenv.Load(".env")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown handler
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		log.Println("🛑 Gracefully shutting down...")
		cancel()
	}()

	// PostgreSQL setup
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		log.Fatal("POSTGRES_URL not set in .env")
	}
	pool, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	// Kafka setup
	kafkaBrokers := os.Getenv("KAFKA_URLS")
	if kafkaBrokers == "" {
		log.Fatal("KAFKA_URLS not set in .env")
	}
	brokers := strings.Split(kafkaBrokers, ",")

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(brokers, ","),
		"group.id":          "chat-db-writers",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("❌ Kafka consumer error: %v", err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{"chat-conversation"}, nil)
	if err != nil {
		log.Fatalf("❌ Kafka subscription error: %v", err)
	}

	log.Println("📥 chat-db-writer service started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("✅ Shutdown complete.")
			return
		default:
			msg, err := consumer.ReadMessage(200 * time.Millisecond)
			if err != nil {
				continue
			}

			var chat Message
			if err := json.Unmarshal(msg.Value, &chat); err != nil {
				log.Printf("❌ Failed to unmarshal Kafka message: %v", err)
				continue
			}

			if err := insertMessage(ctx, pool, &chat); err != nil {
				log.Printf("❌ Failed to insert into DB: %v", err)
			} else {
				log.Printf("✅ Message inserted: [conversation_id=%d]", chat.ConversationID)
			}
		}
	}
}

func insertMessage(ctx context.Context, pool *pgxpool.Pool, msg *Message) error {
	query := `
		INSERT INTO messages (conversation_id, sender_id, text, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := pool.Exec(ctx, query, msg.ConversationID, msg.SenderID, msg.Text, msg.CreatedAt)
	return err
}
