package repository

import (
	"context"
	"errors"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type ChatRepository struct {
	Log      *zap.Logger
	DB       *pgxpool.Pool
	DBCache  *redis.ClusterClient
	Producer *kafka.Producer
	Consumer *kafka.Consumer
}

func NewChatRepository(zap *zap.Logger, db *pgxpool.Pool, dbCache *redis.ClusterClient, kafkaProducer *kafka.Producer, kafkaConsumer *kafka.Consumer) *ChatRepository {
	return &ChatRepository{
		Log:      zap,
		DB:       db,
		DBCache:  dbCache,
		Producer: kafkaProducer,
		Consumer: kafkaConsumer,
	}
}

func (repository *ChatRepository) GetPreviousMessageWithChatID(ctx context.Context, conversationID int, beforeID int, limit int, errorMap map[string]string) ([]model.Message, map[string]string) {
	query := "SELECT id, sender_id, text, created_at FROM messages WHERE conversation_id = $1 AND id < $2 ORDER BY id DESC LIMIT $3"

	var messages []model.Message

	rows, err := repository.DB.Query(ctx, query, conversationID, beforeID, limit)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return messages, errorMap
	}

	defer rows.Close()

	hasData := false

	for rows.Next() {
		var message model.Message
		err = rows.Scan(&message.ID, &message.SenderID, &message.Text, &message.CreatedAt)
		if err != nil {
			errorMap["internal"] = "failed to scan query result"
			return messages, errorMap
		}

		hasData = true
		messages = append(messages, message)
	}

	if hasData == false {
		errorMap["chat"] = "chat not found"
		return messages, errorMap
	}

	return messages, nil
}

func (repository *ChatRepository) GetPreviousMessage(ctx context.Context, conversationID int, limit int, errorMap map[string]string) ([]model.Message, map[string]string) {
	query := "SELECT id, sender_id, text, created_at FROM messages WHERE conversation_id = $1 ORDER BY id DESC LIMIT $2"

	var messages []model.Message

	rows, err := repository.DB.Query(ctx, query, conversationID, limit)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return messages, errorMap
	}

	defer rows.Close()

	hasData := false

	for rows.Next() {
		var message model.Message
		err = rows.Scan(&message.ID, &message.SenderID, &message.Text, &message.CreatedAt)
		if err != nil {
			errorMap["internal"] = "failed to scan query result"
			return messages, errorMap
		}

		hasData = true
		messages = append(messages, message)
	}

	if hasData == false {
		errorMap["chat"] = "chat not found"
		return messages, errorMap
	}

	return messages, nil
}

func (repository *ChatRepository) VerifyAllUserID(ctx context.Context, tx pgx.Tx, allParticipants []string, errorMap map[string]string) map[string]string {
	query := "SELECT id FROM users WHERE id = ANY($1)"
	rows, err := tx.Query(ctx, query, allParticipants)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return errorMap
	}

	defer rows.Close()

	foundUsers := map[string]bool{}
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			errorMap["internal"] = "failed to scan query result"
			return errorMap
		}
		foundUsers[id] = true
	}

	for _, id := range allParticipants {
		if !foundUsers[id] {
			errorMap["participant_ids"] = "one or more participant IDs do not exist"
			return errorMap
		}
	}

	return nil
}

func (repository *ChatRepository) GetConversationIDByParticipants(ctx context.Context, tx pgx.Tx, allParticipants []string, errorMap map[string]string) (int, map[string]string) {
	query := `
	SELECT cp.conversation_id
	FROM conversation_participants cp
	WHERE cp.user_id = ANY($1)
	GROUP BY cp.conversation_id
	HAVING COUNT(*) = $2
	   AND COUNT(*) = (
		 SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = cp.conversation_id
	   )
	LIMIT 1;
	`

	var conversationID int

	err := tx.QueryRow(ctx, query, allParticipants, len(allParticipants)).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		query = "INSERT INTO conversations (created_at) VALUES (NOW()) RETURNING id"
		err = tx.QueryRow(ctx, query).Scan(&conversationID)
		if err != nil {
			errorMap["internal"] = "failed to query into database"
			return conversationID, errorMap
		}

		batch := &pgx.Batch{}
		for _, id := range allParticipants {
			batch.Queue("INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)", conversationID, id)
		}

		br := tx.SendBatch(ctx, batch)
		defer br.Close()
		for i := 0; i < len(allParticipants); i++ {
			_, err = br.Exec()
			if err != nil {
				errorMap["internal"] = "failed to query into database"
				return conversationID, errorMap
			}
		}
	} else if err != nil {
		errorMap["internal"] = "failed to query into database"
		return conversationID, errorMap
	}

	return conversationID, nil
}

func (repository *ChatRepository) GetParticipantID(ctx context.Context, tx pgx.Tx, userUUID string, conversationID int, errorMap map[string]string) (string, map[string]string) {
	query := "SELECT user_id FROM conversation_participants WHERE conversation_id=$1 AND user_id!=$2"

	var participantID string
	err := tx.QueryRow(ctx, query, conversationID, userUUID).Scan(&participantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["conversation_id"] = "conversation not found"
			return participantID, errorMap
		} else {
			errorMap["internal"] = "failed to query into database"
			return participantID, errorMap
		}
	}

	return participantID, nil
}

func (repository *ChatRepository) GetParticipantName(ctx context.Context, tx pgx.Tx, participationID string, errorMap map[string]string) (string, map[string]string) {
	query := "SELECT username FROM users WHERE id=$1"

	var participantName string
	err := tx.QueryRow(ctx, query, participationID).Scan(&participantName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["conversation_id"] = "conversation not found"
			return participantName, errorMap
		} else {
			errorMap["internal"] = "failed to query into database"
			return participantName, errorMap
		}
	}

	return participantName, nil
}

func (repository *ChatRepository) SetWSToken(ctx context.Context, userUUID string, wsToken string, duration time.Duration, errorMap map[string]string) map[string]string {
	err := repository.DBCache.Set(ctx, "ws_token:"+wsToken, userUUID, duration).Err()
	if err != nil {
		errorMap["internal"] = "failed to set key in redis db"
		return errorMap
	}

	return nil
}

func (repository *ChatRepository) InsertMessage(ctx context.Context, msg model.Message, errorMap map[string]string) (int, map[string]string) {
	var id int
	query := `INSERT INTO messages (conversation_id, sender_id, text, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	err := repository.DB.QueryRow(ctx, query, msg.ConversationID, msg.SenderID, msg.Text, msg.CreatedAt).Scan(&id)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return id, errorMap
	}

	return id, nil
}

func (repository *ChatRepository) GetParticipants(ctx context.Context, conversationID int) ([]string, map[string]string) {
	errorMap := map[string]string{}

	query := "SELECT user_id FROM conversation_participants WHERE conversation_id = $1"
	rows, err := repository.DB.Query(ctx, query, conversationID)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return nil, errorMap
	}

	defer rows.Close()

	hasData := false

	var participantIDs []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			errorMap["internal"] = "failed to scan query result"
			return nil, errorMap
		}

		hasData = true
		participantIDs = append(participantIDs, id)
	}

	if hasData == false {
		errorMap["user"] = "user not found"
		return nil, errorMap
	}

	return participantIDs, nil
}

func (repository *ChatRepository) CheckUserExistence(ctx context.Context, userUUID string, errorMap map[string]string) map[string]string {
	query := "SELECT username FROM users WHERE id=$1"

	var username string
	err := repository.DB.QueryRow(ctx, query, userUUID).Scan(&username)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["user"] = "user not found"
			return errorMap
		}
		errorMap["internal"] = "failed to query database"
		return errorMap
		//repository.Log.Panic("failed to query database", zap.Error(err))
	}

	return nil
}

func (repository *ChatRepository) VerifyWsToken(ctx context.Context, wsToken string, errorMap map[string]string) (string, map[string]string) {
	userUUID, err := repository.DBCache.Get(ctx, "ws_token:"+wsToken).Result()
	if err == redis.Nil {
		errorMap["auth"] = "invalid or expired ws token"
		return "", errorMap
	} else if err != nil {
		errorMap["internal"] = "failed to get into redis"
		return "", errorMap
	}

	repository.DBCache.Del(ctx, "ws_token:"+wsToken)

	return userUUID, nil
}

func (repository *ChatRepository) SAddConversationMember(ctx context.Context, userUUID string, conversationID int) {
	key := "conversation:" + strconv.Itoa(conversationID) + ":participants"
	repository.DBCache.SAdd(ctx, key, userUUID)
}

func (repository *ChatRepository) SetUserSession(ctx context.Context, userUUID string) {
	repository.DBCache.Set(ctx, "user:"+userUUID+":conn", "active", time.Minute)
	repository.DBCache.Expire(ctx, "user:"+userUUID+":status", time.Minute)
}

//func (repository *ChatRepository) ConsumeConversationMessages(ctx context.Context, conversationID int, userUUID string, conn *websocket.Conn) {
//	topic := "chat-conversation" + strconv.Itoa(conversationID)
//	err := repository.Consumer.SubscribeTopics([]string{topic}, nil)
//	if err != nil {
//		return
//	}
//
//	for {
//		msg, err := repository.Consumer.ReadMessage(-1)
//		if err != nil {
//			break
//		}
//
//		_ = conn.WriteMessage(websocket.TextMessage, msg.Value)
//		repository.SetUserSession(ctx, userUUID)
//	}
//}

func (repository *ChatRepository) SubscribeToRedisChannel(ctx context.Context, channel string) *redis.PubSub {
	return repository.DBCache.Subscribe(ctx, channel)
}

func (repository *ChatRepository) ProduceToKafka(ctx context.Context, topic string, message []byte) error {
	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	err := repository.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)

	if err != nil {
		return err
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)
	return m.TopicPartition.Error
}

func (r *ChatRepository) GetConversationParticipantsByConversationID(ctx context.Context, conversationID int, errorMap map[string]string) ([]string, map[string]string) {
	query := `
		SELECT user_id 
		FROM conversation_participants 
		WHERE conversation_id = $1
	`

	rows, err := r.DB.Query(ctx, query, conversationID)
	if err != nil {
		errorMap["internal"] = "failed to query database"
		return nil, errorMap
	}
	defer rows.Close()

	hasData := false

	var participants []string
	for rows.Next() {
		var userID string
		err = rows.Scan(&userID)
		if err != nil {
			errorMap["internal"] = "failed to scan query result"
			return nil, errorMap
		}

		hasData = true
		participants = append(participants, userID)
	}

	if hasData == false {
		errorMap["user"] = "user not found"
		return nil, errorMap
	}

	return participants, nil
}
