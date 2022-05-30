package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/G-Research/armada/internal/pulsarutils"
)

// BatchUpdate represents an Event Row along with information about the originating pulsar message
type BatchUpdate struct {
	MessageIds []*pulsarutils.ConsumerMessageId
	Events     []*Event
}

type Event struct {
	Queue  string
	Jobset string
	Event  []byte
}

// ExternalSeqNo is a sequence number that we pass to end users
// Sequence is the puslar message sequence
// Index is the index of the event inside the armadaevents.EventSequence
type ExternalSeqNo struct {
	Sequence int64
	Index    int
}

// ParseExternalSeqNo Parses an external sequence number which should be of the form "Sequence:Index".
// The empty string will be interpreted as "-1:-1" whoch is the initial sequence numebr
// An error will be returned if the sequence number cannot be parsed
func ParseExternalSeqNo(str string) (*ExternalSeqNo, error) {
	if str == "" {
		return &ExternalSeqNo{-1, -1}, nil
	}
	toks := strings.Split(str, ":")
	if len(toks) != 2 {
		return nil, fmt.Errorf("%s is not a valid sequence number", str)
	}
	sequence, err := strconv.ParseInt(toks[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid sequence number", str)
	}
	index, err := strconv.Atoi(toks[1])
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid sequence number", str)
	}
	return &ExternalSeqNo{
		Sequence: sequence,
		Index:    index,
	}, nil
}

// IsValidExternalSeqNo Returns true if the given string is a valid ExternalSeqNo
func IsValidExternalSeqNo(str string) bool {
	_, err := ParseExternalSeqNo(str)
	return err == nil
}

func (e *ExternalSeqNo) ToString() string {
	return fmt.Sprintf("%d:%d", e.Sequence, e.Index)
}

func (e *ExternalSeqNo) IsAfter(other *ExternalSeqNo) bool {
	if other == nil {
		return false
	}
	if e.Sequence > other.Sequence {
		return true
	}
	if e.Sequence == other.Sequence && e.Index > other.Index {
		return true
	}
	return false
}
