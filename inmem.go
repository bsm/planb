package planb

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

var errInvalidStorageKey = errors.New("planb: invalid storage key")

const numInMemShards = 64

type InmemStore struct {
	shards [numInMemShards]*inMemShard
}

// NewInmemStore opens a new simplistic, non-transactional in-memory KVStore
func NewInmemStore() *InmemStore {
	store := new(InmemStore)
	for i := 0; i < numInMemShards; i++ {
		store.shards[i] = &inMemShard{data: make(map[string][]byte)}
	}
	return store
}

// Get retrieves a key
func (s *InmemStore) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errInvalidStorageKey
	}

	val := s.shards[fnv32a(key)%numInMemShards].Get(key)
	return val, nil
}

// Put sets a key
func (s *InmemStore) Put(key, val []byte) error {
	if len(key) == 0 {
		return errInvalidStorageKey
	}
	s.shards[fnv32a(key)%numInMemShards].Put(key, val)
	return nil
}

// Delete deletes a key
func (s *InmemStore) Delete(key []byte) error {
	return s.Put(key, nil)
}

// Snapshot implements Store
func (s *InmemStore) Snapshot(w io.Writer) error {
	buf := make([]byte, binary.MaxVarintLen64)
	for i := 0; i < numInMemShards; i++ {
		if err := s.shards[i].Snapshot(buf, w); err != nil {
			return err
		}
	}
	return nil
}

// Restore implements Store
func (s *InmemStore) Restore(r io.Reader) error {
	snap := &inMemSnapshotIterator{Reader: bufio.NewReader(r)}
	for {
		err := snap.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if err := s.Put(snap.key, snap.val); err != nil {
			return err
		}
	}
}

// --------------------------------------------------------------------

type inMemShard struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func (s *inMemShard) Snapshot(buf []byte, w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for key, val := range s.data {
		n := binary.PutUvarint(buf[:binary.MaxVarintLen64], uint64(len(key)))
		if _, err := w.Write(buf[:n]); err != nil {
			return err
		}
		if _, err := io.WriteString(w, key); err != nil {
			return err
		}

		n = binary.PutUvarint(buf[:binary.MaxVarintLen64], uint64(len(val)))
		if _, err := w.Write(buf[:n]); err != nil {
			return err
		}
		if _, err := w.Write(val); err != nil {
			return err
		}
	}
	return nil
}

func (s *inMemShard) Get(key []byte) []byte {
	s.mu.RLock()
	val := s.data[string(key)]
	s.mu.RUnlock()
	return val
}

func (s *inMemShard) Put(key, val []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if val == nil {
		delete(s.data, string(key))
	} else {
		s.data[string(key)] = val
	}
}

// --------------------------------------------------------------------

type inMemSnapshotIterator struct {
	*bufio.Reader
	key, val []byte
}

func (s *inMemSnapshotIterator) Next() error {
	u, err := binary.ReadUvarint(s)
	if err != nil {
		return err
	}
	s.key = make([]byte, int(u))
	if _, err := io.ReadFull(s, s.key); err != nil {
		return err
	}

	u, err = binary.ReadUvarint(s)
	if err != nil {
		return err
	}
	s.val = make([]byte, int(u))
	if _, err := io.ReadFull(s, s.val); err != nil {
		return err
	}
	return nil
}
