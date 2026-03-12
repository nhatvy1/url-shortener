package services

import (
	"sync"
	"time"
)

type Snowflake struct {
	mu            sync.Mutex
	lastTimestamp int64
	sequence      int64
	nodeID        int64
}

func NewSnowflake(nodeID int64) *Snowflake {
	return &Snowflake{
		nodeID: nodeID,
	}
}

func (s *Snowflake) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().UnixMilli()
	if timestamp == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & 0xFFF // 12 bits for sequence
		if s.sequence == 0 {
			for timestamp <= s.lastTimestamp {
				timestamp = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}
	s.lastTimestamp = timestamp

	id := ((timestamp - 1288834974657) << 22) | (s.nodeID << 12) | s.sequence
	return id
}
