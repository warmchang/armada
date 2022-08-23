package repository

import (
	"time"

	"github.com/G-Research/armada/internal/armada/repository/sequence"

	"github.com/G-Research/armada/internal/common/compress"

	"github.com/go-redis/redis"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/G-Research/armada/internal/eventapi/serving/apimessages"
	"github.com/G-Research/armada/pkg/api"
	"github.com/G-Research/armada/pkg/armadaevents"
)

const eventStreamPrefix = "Events:"
const dataKey = "message"

type EventRepository interface {
	CheckStreamExists(queue string, jobSetId string) (bool, error)
	ReadEvents(queue, jobSetId string, lastId string, limit int64, block time.Duration) ([]*api.EventStreamMessage, error)
	GetLastMessageId(queue, jobSetId string) (string, error)
}

type RedisEventRepository struct {
	db redis.UniversalClient
}

func (repo *RedisEventRepository) ReadEvents(queue string, jobSetId string, from *sequence.ExternalSeqNo, limit int64, block time.Duration) ([]byte, error) {

	cmd, err := repo.db.XRead(&redis.XReadArgs{
		Streams: []string{repo.getJobSetEventsKey(queue, jobSetId), from.PrevRedisId()},
		Count:   limit,
		Block:   block,
	}).Result()

	// redis signals empty list by Nil
	if err == redis.Nil {
		return make([]*api.EventStreamMessage, 0), nil
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	//TODO: find some way of sharing the decompressor
	decompressor, err := compress.NewZlibDecompressor()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	messages := make([]*api.EventStreamMessage, 0)
	for _, m := range cmd[0].Messages {
		data := m.Values[dataKey]
		bytes := []byte(data.(string))
		// TODO: here we decompress all the events we fetched from the db- it would be much better
		// If we could decompress lazily, but the interface confines us somewhat here
		apiEvents, err := extractEvents(bytes, queue, jobSetId, decompressor)
		if err != nil {
			return nil, err
		}
		for i, msg := range apiEvents {
			msgId, err := sequence.FromRedisId(m.ID, i, i == len(apiEvents)-1)
			if err != nil {
				return nil, err
			}
			messages = append(messages, &api.EventStreamMessage{Id: msgId.ToString(), Message: msg})
		}
	}
	return messages, nil
}

func extractEvents(data []byte, queue, jobSetId string, decompressor *compress.ZlibDecompressor) ([]*api.EventMessage, error) {
	decompressedData, err := decompressor.Decompress(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	es := &armadaevents.EventSequence{}
	err = proto.Unmarshal(decompressedData, es)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// These fields are not present in the db messages, so we add them back here
	es.Queue = queue
	es.JobSetName = jobSetId
	return apimessages.FromEventSequence(es)
}

func (repo *RedisEventRepository) GetLastMessageId(queue, jobSetId string) (string, error) {
	msg, err := repo.db.XRevRangeN(getJobSetEventsKey(queue, jobSetId), "+", "-", 1).Result()
	if err != nil {
		return "", errors.Wrap(err, "Error retrieving the last message id from Redis")
	}
	if len(msg) > 0 {
		data := msg[0].Values[dataKey]
		bytes := []byte(data.(string))
		// TODO: here we decompress all the events we fetched from the db- it would be much better
		// If we could decompress lazily, but the interface confines us somewhat here
		apiEvents, err := extractEvents(bytes, queue, jobSetId, decompressor)
		if err != nil {
			return nil, err
		}
		for i, msg := range apiEvents {
			msgId, err := sequence.FromRedisId(m.ID, i, i == len(apiEvents)-1)
			if err != nil {
				return nil, err
			}
			messages = append(messages, &api.EventStreamMessage{Id: msgId.ToString(), Message: msg})
		}
	}
	return "0", nil
}

func (repo *RedisEventRepository) getJobSetEventsKey(queue, jobSetId string) string {
	return eventStreamPrefix + queue + ":" + jobSetId
}
