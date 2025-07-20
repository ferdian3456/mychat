package config

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

func NewKafkaConsumer(config *koanf.Koanf, log *zap.Logger) *kafka.Consumer {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.String("KAFKA_URLS"),
		"group.id":          "my-consumer-group",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		log.Fatal("Failed to connect redis", zap.Error(err))
	}

	return consumer
}

func NewKafkaProducer(config *koanf.Koanf, log *zap.Logger) *kafka.Producer {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.String("KAFKA_URLS"),
	})

	if err != nil {
		log.Fatal("Failed to connect redis", zap.Error(err))
	}

	return producer
}
