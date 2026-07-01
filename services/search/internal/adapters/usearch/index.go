package usearch

import (
	"fmt"
	"sync"

	usearch "github.com/unum-cloud/usearch/golang"

	"github.com/marketplace/search/internal/domain"
)

const vectorDimensions = 384

type Index struct {
	mu    sync.RWMutex
	index *usearch.Index
}

func NewIndex() (*Index, error) {
	conf := usearch.IndexConfig{
		Metric:          usearch.InnerProduct,
		Dimensions:      vectorDimensions,
		Quantization:    usearch.F32,
		Connectivity:    32,
		ExpansionAdd:    128,
		ExpansionSearch: 64,
	}

	idx, err := usearch.NewIndex(conf)
	if err != nil {
		return nil, fmt.Errorf("create usearch index: %w", err)
	}

	return &Index{index: idx}, nil
}

func (u *Index) Add(key uint64, vector []float32) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.index.Add(key, vector)
}

func (u *Index) Search(vector []float32, k int) ([]domain.Neighbor, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	keys, distances, err := u.index.Search(vector, uint(k))
	if err != nil {
		return nil, fmt.Errorf("usearch search: %w", err)
	}

	results := make([]domain.Neighbor, len(keys))
	for i := range keys {
		results[i] = domain.Neighbor{
			ID:       int64(keys[i]),
			Distance: distances[i],
			Score:    distanceToScore(distances[i]),
		}
	}
	return results, nil
}

// distanceToScore converts the USearch InnerProduct distance into a similarity
// score in [0, 1] where 1 means (nearly) identical.
//
// For L2-normalized vectors the inner product equals the cosine similarity in
// [-1, 1]. USearch reports the IP metric as distance = 1 - ip, so we recover
// ip = 1 - distance and linearly remap [-1, 1] -> [0, 1].
func distanceToScore(distance float32) float32 {
	ip := 1 - distance
	score := (ip + 1) / 2
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func (u *Index) Len() int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	n, _ := u.index.Len()
	return int(n)
}

func (u *Index) Reset() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.index.Clear()
}

func (u *Index) Reserve(capacity int) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.index.Reserve(uint(capacity))
}

func (u *Index) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.index.Destroy()
}
