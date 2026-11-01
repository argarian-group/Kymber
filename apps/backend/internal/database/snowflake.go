package database

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"
)

func NewShortID() (string, error) {
	value := make([]byte, 3)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

const snowflakeEpoch = int64(1704067200000)

var snowflake struct {
	sync.Mutex
	timestamp int64
	sequence  uint16
}

func NewID() uint64 {
	snowflake.Lock()
	defer snowflake.Unlock()
	now := max(time.Now().UnixMilli()-snowflakeEpoch, snowflake.timestamp)
	if now == snowflake.timestamp {
		snowflake.sequence = (snowflake.sequence + 1) & 0x0fff
		if snowflake.sequence == 0 {
			for now <= snowflake.timestamp {
				now = time.Now().UnixMilli() - snowflakeEpoch
			}
		}
	} else {
		var node [2]byte
		if _, err := rand.Read(node[:]); err != nil {
			panic(err)
		}
		snowflake.sequence = binary.BigEndian.Uint16(node[:]) & 0x0fff
	}
	snowflake.timestamp = now
	return uint64(now)<<22 | uint64(snowflake.sequence)
}
