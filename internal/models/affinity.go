package models

import (
	"container/list"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// AffinityEntry 会话亲和条目。
// ClientID 为众包 clientID 或 "direct:"+backendID；IsDirect 标记 Direct 后端。
type AffinityEntry struct {
	ClientID   string // 众包 clientID 或 "direct:"+backendID
	IsDirect   bool
	ExpireAt   int64 // unix ns，滑动 TTL
	LeaseUntil int64 // unix ns，0=无租约；粘住的目标临时不可用时保留等待回迁
	MissCount  int32 // 连续实测低命中次数
}

// AffinityStats 亲和聚合计数快照（供每分钟聚合日志/监控）。
// 计数为自上次 Stats() 读取以来的增量（读取即清零）。
type AffinityStats struct {
	Hits        int64 // 亲和命中（Get 返回 true）
	Rebinds     int64 // 回迁/换目标（Touch 时 ClientID 变化）
	Evictions   int64 // 逐出（LRU 超限 + 过期惰性删除 + janitor 清理）
	MissDeletes int64 // 实测失效删除（RecordMiss 达到 K 后调用方 Delete）
}

// AffinityStore 会话亲和存储接口。P4 换 Redis 实现时接口不变。
type AffinityStore interface {
	Get(key string) (AffinityEntry, bool)      // 过期即 miss（惰性删除）
	Touch(key, clientID string, isDirect bool) // 写入/续期；换目标时 MissCount/Lease 清零
	MarkLease(key string)                      // 目标临时不可用：设置 LeaseUntil=now+lease，不删
	LeaseExpired(key string) bool              // 无记录或 LeaseUntil 过期 → true
	RecordMiss(key string) int32               // MissCount++，返回新值（达到 K 由调用方 Delete）
	Delete(key string)
	Len() int             // 监控/测试
	Stats() AffinityStats // 读取并清零增量计数
	Close()               // 停止后台 janitor（测试用）
}

const affinityShards = 32

type affinityNode struct {
	key   string
	entry AffinityEntry
}

type affinityShard struct {
	mu  sync.Mutex
	m   map[string]*list.Element
	lru *list.List // 队头最新，队尾最旧
}

type memoryAffinityStore struct {
	shards      [affinityShards]*affinityShard
	ttl         time.Duration
	lease       time.Duration
	maxPerShard int
	done        chan struct{}

	// 聚合计数（原子，Stats() 读取即清零）
	hits        atomic.Int64
	rebinds     atomic.Int64
	evictions   atomic.Int64
	missDeletes atomic.Int64
}

// NewMemoryAffinityStore 创建内存亲和存储。ttl 为滑动过期，lease 为回迁租约时长。
func NewMemoryAffinityStore(ttl, lease time.Duration, maxEntries int) AffinityStore {
	if ttl <= 0 {
		ttl = 20 * time.Minute
	}
	if lease <= 0 {
		lease = 60 * time.Second
	}
	if maxEntries <= 0 {
		maxEntries = 100000
	}
	perShard := maxEntries / affinityShards
	if perShard < 1 {
		perShard = 1
	}
	s := &memoryAffinityStore{
		ttl:         ttl,
		lease:       lease,
		maxPerShard: perShard,
		done:        make(chan struct{}),
	}
	for i := 0; i < affinityShards; i++ {
		s.shards[i] = &affinityShard{
			m:   make(map[string]*list.Element),
			lru: list.New(),
		}
	}
	go s.janitor()
	return s
}

func (s *memoryAffinityStore) shardFor(key string) *affinityShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.shards[h.Sum32()&(affinityShards-1)]
}

func (s *memoryAffinityStore) Close() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *memoryAffinityStore) janitor() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			now := time.Now().UnixNano()
			for _, sh := range s.shards {
				sh.mu.Lock()
				for e := sh.lru.Back(); e != nil; {
					prev := e.Prev()
					n := e.Value.(*affinityNode)
					if n.entry.ExpireAt <= now {
						delete(sh.m, n.key)
						sh.lru.Remove(e)
						s.evictions.Add(1)
					}
					e = prev
				}
				sh.mu.Unlock()
			}
		}
	}
}

func (s *memoryAffinityStore) Get(key string) (AffinityEntry, bool) {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	e, ok := sh.m[key]
	if !ok {
		return AffinityEntry{}, false
	}
	n := e.Value.(*affinityNode)
	if n.entry.ExpireAt <= time.Now().UnixNano() {
		delete(sh.m, key)
		sh.lru.Remove(e)
		s.evictions.Add(1)
		return AffinityEntry{}, false
	}
	sh.lru.MoveToFront(e)
	s.hits.Add(1)
	return n.entry, true
}

func (s *memoryAffinityStore) Touch(key, clientID string, isDirect bool) {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	now := time.Now().UnixNano()
	if e, ok := sh.m[key]; ok {
		n := e.Value.(*affinityNode)
		// 换目标时 MissCount/Lease 清零
		if n.entry.ClientID != clientID {
			n.entry.MissCount = 0
			n.entry.LeaseUntil = 0
			s.rebinds.Add(1)
		}
		n.entry.ClientID = clientID
		n.entry.IsDirect = isDirect
		n.entry.ExpireAt = now + s.ttl.Nanoseconds()
		sh.lru.MoveToFront(e)
		return
	}
	// 新增
	n := &affinityNode{
		key: key,
		entry: AffinityEntry{
			ClientID: clientID,
			IsDirect: isDirect,
			ExpireAt: now + s.ttl.Nanoseconds(),
		},
	}
	e := sh.lru.PushFront(n)
	sh.m[key] = e
	// LRU 逐出
	for sh.lru.Len() > s.maxPerShard {
		oldest := sh.lru.Back()
		if oldest == nil {
			break
		}
		on := oldest.Value.(*affinityNode)
		delete(sh.m, on.key)
		sh.lru.Remove(oldest)
		s.evictions.Add(1)
	}
}

func (s *memoryAffinityStore) MarkLease(key string) {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if e, ok := sh.m[key]; ok {
		n := e.Value.(*affinityNode)
		n.entry.LeaseUntil = time.Now().UnixNano() + s.lease.Nanoseconds()
	}
}

func (s *memoryAffinityStore) LeaseExpired(key string) bool {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	e, ok := sh.m[key]
	if !ok {
		return true
	}
	n := e.Value.(*affinityNode)
	return n.entry.LeaseUntil <= time.Now().UnixNano()
}

func (s *memoryAffinityStore) RecordMiss(key string) int32 {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if e, ok := sh.m[key]; ok {
		n := e.Value.(*affinityNode)
		n.entry.MissCount++
		return n.entry.MissCount
	}
	return 0
}

func (s *memoryAffinityStore) Delete(key string) {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if e, ok := sh.m[key]; ok {
		delete(sh.m, key)
		sh.lru.Remove(e)
		s.missDeletes.Add(1)
	}
}

// Stats 返回并清零自上次读取以来的增量计数。
func (s *memoryAffinityStore) Stats() AffinityStats {
	return AffinityStats{
		Hits:        s.hits.Swap(0),
		Rebinds:     s.rebinds.Swap(0),
		Evictions:   s.evictions.Swap(0),
		MissDeletes: s.missDeletes.Swap(0),
	}
}

func (s *memoryAffinityStore) Len() int {
	total := 0
	for _, sh := range s.shards {
		sh.mu.Lock()
		total += len(sh.m)
		sh.mu.Unlock()
	}
	return total
}
