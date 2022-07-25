package repository

import (
	"fmt"
	"time"

	"github.com/G-Research/armada/internal/armada/configuration"
	"github.com/G-Research/armada/pkg/api"
	"github.com/go-redis/redis"
	"github.com/gogo/protobuf/proto"
)

type EventStore interface {
	ReportEvents(message []*api.EventMessage) error
}

type LegacyRedisEventRepository struct {
	db             redis.UniversalClient
	eventRetention configuration.EventRetentionPolicy
}

func NewLegacyRedisEventRepository(db redis.UniversalClient, eventRetention configuration.EventRetentionPolicy) *LegacyRedisEventRepository {
	return &LegacyRedisEventRepository{db: db, eventRetention: eventRetention}
}

func (repo *LegacyRedisEventRepository) ReportEvent(message *api.EventMessage) error {
	return repo.ReportEvents([]*api.EventMessage{message})
}

func (repo *LegacyRedisEventRepository) ReportEvents(messages []*api.EventMessage) error {
	if len(messages) == 0 {
		return nil
	}

	type eventData struct {
		key  string
		data []byte
	}
	data := []eventData{}
	uniqueJobSets := make(map[string]bool)

	for _, m := range messages {
		event, e := api.UnwrapEvent(m)
		if e != nil {
			return e
		}
		messageData, e := proto.Marshal(m)
		if e != nil {
			return e
		}
		key := getJobSetEventsKey(event.GetQueue(), event.GetJobSetId())
		data = append(data, eventData{key: key, data: messageData})
		uniqueJobSets[key] = true
	}

	pipe := repo.db.Pipeline()
	for _, e := range data {
		pipe.XAdd(&redis.XAddArgs{
			Stream: e.key,
			Values: map[string]interface{}{
				dataKey: e.data,
			},
		})
	}

	if repo.eventRetention.ExpiryEnabled {
		for key := range uniqueJobSets {
			pipe.Expire(key, repo.eventRetention.RetentionDuration)
		}
	}

	_, e := pipe.Exec()
	return e
}

func (repo *LegacyRedisEventRepository) CheckStreamExists(queue string, jobSetId string) (bool, error) {
	result, err := repo.db.Exists(getJobSetEventsKey(queue, jobSetId)).Result()
	if err != nil {
		return false, err
	}
	exists := result > 0
	return exists, nil
}

func (repo *LegacyRedisEventRepository) ReadEvents(queue, jobSetId string, lastId string, limit int64, block time.Duration) ([]*api.EventStreamMessage, error) {

	if lastId == "" {
		lastId = "0"
	}

	cmd, err := repo.db.XRead(&redis.XReadArgs{
		Streams: []string{getJobSetEventsKey(queue, jobSetId), lastId},
		Count:   limit,
		Block:   block,
	}).Result()

	// redis signals empty list by Nil
	if err == redis.Nil {
		return make([]*api.EventStreamMessage, 0), nil
	} else if err != nil {
		return nil, fmt.Errorf("[LegacyRedisEventRepository.ReadEvents] error reading from database: %s", err)
	}

	messages := make([]*api.EventStreamMessage, 0)
	for _, m := range cmd[0].Messages {
		data := m.Values[dataKey]
		msg := &api.EventMessage{}
		bytes := []byte(data.(string))
		err = proto.Unmarshal(bytes, msg)
		if err != nil {
			return nil, fmt.Errorf("[LegacyRedisEventRepository.ReadEvents] error unmarshalling: %s", err)
		}
		messages = append(messages, &api.EventStreamMessage{Id: m.ID, Message: msg})
	}
	return messages, nil
}

func (repo *LegacyRedisEventRepository) GetLastMessageId(queue, jobSetId string) (string, error) {
	msg, err := repo.db.XRevRangeN(getJobSetEventsKey(queue, jobSetId), "+", "-", 1).Result()
	if err != nil {
		return "", fmt.Errorf("[LegacyRedisEventRepository.GetLastMessageId] error reading from database: %s", err)
	}
	if len(msg) > 0 {
		return msg[0].ID, nil
	}
	return "0", nil
}

func getJobSetEventsKey(queue, jobSetId string) string {
	return eventStreamPrefix + queue + ":" + jobSetId
}
