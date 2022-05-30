package ingestion

import (
	"context"
	"time"

	"github.com/G-Research/armada/internal/common/armadaerrors"

	"github.com/G-Research/armada/internal/common/database"

	log "github.com/sirupsen/logrus"

	"github.com/G-Research/armada/internal/eventapi/eventdb"
	"github.com/G-Research/armada/internal/eventapi/model"
)

// InsertEvents takes a channel of armada events and insets them into the event db
// the events are republished to an output channel for further processing (e.g. Ackking)
func InsertEvents(ctx context.Context, db *eventdb.RedisEventStore, msgs chan *model.BatchUpdate, bufferSize int) chan *model.BatchUpdate {
	out := make(chan *model.BatchUpdate, bufferSize)
	go func() {
		for msg := range msgs {
			insert(db, msg.Events)
			out <- msg
		}
		close(out)
	}()
	return out
}

func insert(db *eventdb.RedisEventStore, rows []*model.Event) error {
	start := time.Now()

	var inserted = false

	for !inserted {
		err := db.ReportEvents(rows)
		if err == nil {
			return result, nil
		}

		if armadaerrors.IsNetworkError(err) || armadaerrors.IsRetryablePostgresError(err) {
			backOff = min(2*backOff, maxBackoff)
			numRetries++
			log.WithError(err).Warnf("Retryable error encountered executing sql, will wait for %d seconds before retrying", backOff)
			time.Sleep(time.Duration(backOff) * time.Second)
		} else {
			// Non retryable error
			return nil, err
		}
	}

	err := database.ExecuteWithDatabaseRetry(func() error {
		return
	})
	if err != nil {
		log.WithError(err).Warnf("Error inserting rows")
	} else {
		taken := time.Now().Sub(start).Milliseconds()
		log.Infof("Inserted %d events in %dms", len(rows), taken)
	}
}
