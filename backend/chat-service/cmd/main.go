package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/model"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load("./.env")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	kafkaBrokers := os.Getenv("KAFKA_URLS")
	if kafkaBrokers == "" {
		log.Fatal("KAFKA_BROKERS not set in .env")
	}

	// Kafka consumer setup
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBrokers,
		"group.id":          "redis-publisher-worker",
		"auto.offset.reset": "latest",
	})
	if err != nil {
		log.Fatalf("Kafka consumer error: %v", err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{"chat-conversation"}, nil)
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	// Redis Cluster setup
	redisURLs := os.Getenv("REDIS_URLS")
	if redisURLs == "" {
		log.Fatal("REDIS_URLS is not set in .env")
	}
	clusterAddrs := strings.Split(redisURLs, ",")

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: clusterAddrs,
	})
	defer rdb.Close()

	log.Println("🚀 Kafka to Redis Cluster publisher started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Shutting down")
			return
		default:
			msg, err := consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				continue
			}

			//log.Printf("📦 Raw Kafka Payload: %s\n", string(msg.Value))

			var chat model.Message
			if err := json.Unmarshal(msg.Value, &chat); err != nil {
				log.Println("❌ Invalid JSON message:", err)
				continue
			}

			//log.Printf("💬 Parsed Message: %+v\n", chat)

			for _, userID := range chat.RecipientIDs {
				bucket := helper.GetBucketForUser(userID, 1024)
				channel := fmt.Sprintf("deliver:bucket:%d", bucket)

				if err := rdb.Publish(ctx, channel, msg.Value).Err(); err != nil {
					log.Printf("❌ Redis publish failed: %v\n", err)
				}
			}
		}
	}
}
