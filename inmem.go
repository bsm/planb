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

type inMemStore struct {
	shards [numInMemShards]*inMemShard
}

// NewInmemStore opens a new simplistic, non-transactional in-memory KVStore
func NewInmemStore() KVStore {
	store := new(inMemStore)
	for i := 0; i < numInMemShards; i++ {
		store.shards[i] = &inMemShard{data: make(map[string][]byte)}
	}
	return store
}

// Close implements KVStore
func (*inMemStore) Close() error { return nil }

// Begin implements KVStore
func (s *inMemStore) Begin(_ bool) (KVBatch, error) {
	return &inMemoryBatch{inMemStore: s}, nil
}

// Get implements KVBatch
func (s *inMemStore) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errInvalidStorageKey
	}

	val := s.shards[fnv32a(key)%numInMemShards].Get(key)
	return val, nil
}

func (s *inMemStore) put(key, val []byte) error {
	if len(key) == 0 {
		return errInvalidStorageKey
	}
	s.shards[fnv32a(key)%numInMemShards].Put(key, val)
	return nil
}

// Snapshot implements KVStore
func (s *inMemStore) Snapshot(w io.Writer) error {
	buf := make([]byte, binary.MaxVarintLen64)
	for i := 0; i < numInMemShards; i++ {
		if err := s.shards[i].Snapshot(buf, w); err != nil {
			return err
		}
	}
	return nil
}

// Restore implements KVStore
func (s *inMemStore) Restore(r io.Reader) error {
	snap := &inMemSnapshotIterator{Reader: bufio.NewReader(r)}
	for {
		err := snap.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if err := s.put(snap.key, snap.val); err != nil {
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

// --------------------------------------------------------------------

type inMemoryBatchWrite struct{ key, val []byte }

type inMemoryBatch struct {
	*inMemStore
	stash []inMemoryBatchWrite
}

// Rollback implements KVBatch
func (b *inMemoryBatch) Rollback() error {
	b.stash = b.stash[:0]
	return nil
}

// Commit implements KVBatch
func (b *inMemoryBatch) Commit() error {
	for _, w := range b.stash {
		if err := b.inMemStore.put(w.key, w.val); err != nil {
			return err
		}
	}
	return nil
}

// Put implements KVBatch
func (b *inMemoryBatch) Put(key, val []byte) error {
	if len(key) == 0 {
		return errInvalidStorageKey
	}
	b.stash = append(b.stash, inMemoryBatchWrite{key: key, val: val})
	return nil
}

// Delete implements KVBatch
func (b *inMemoryBatch) Delete(key []byte) error {
	return b.Put(key, nil)
}
