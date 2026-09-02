package models

import (
	"container/list"
	"fmt"
	"sync"
	"testing"
	"time"
)

func newTestAffinity(ttl, lease time.Duration, maxEntries int) *memoryAffinityStore {
	s := &memoryAffinityStore{
		ttl:         ttl,
		lease:       lease,
		maxPerShard: maxEntries / affinityShards,
		done:        make(chan struct{}),
	}
	if s.maxPerShard < 1 {
		s.maxPerShard = 1
	}
	for i := 0; i < affinityShards; i++ {
		s.shards[i] = &affinityShard{
			m:   make(map[string]*list.Element),
			lru: list.New(),
		}
	}
	return s
}

func TestAffinityTouchGet(t *testing.T) {
	s := newTestAffinity(50*time.Millisecond, time.Second, 1000)
	defer s.Close()

	s.Touch("k1", "c1", false)
	e, ok := s.Get("k1")
	if !ok || e.ClientID != "c1" {
		t.Fatalf("expected hit c1, got %+v ok=%v", e, ok)
	}
	// 滑动续期：touch 后 TTL 重置
	time.Sleep(30 * time.Millisecond)
	s.Touch("k1", "c1", false)
	time.Sleep(30 * time.Millisecond)
	if _, ok := s.Get("k1"); !ok {
		t.Fatalf("expected sliding renewal to keep entry alive")
	}
	// 过期
	time.Sleep(40 * time.Millisecond)
	if _, ok := s.Get("k1"); ok {
		t.Fatalf("expected miss after TTL expiry")
	}
}

func TestAffinityLease(t *testing.T) {
	s := newTestAffinity(time.Minute, 50*time.Millisecond, 1000)
	defer s.Close()

	s.Touch("k1", "c1", false)
	if !s.LeaseExpired("k1") {
		t.Fatalf("no lease set → should be expired")
	}
	s.MarkLease("k1")
	if s.LeaseExpired("k1") {
		t.Fatalf("lease active → should not be expired")
	}
	time.Sleep(60 * time.Millisecond)
	if !s.LeaseExpired("k1") {
		t.Fatalf("lease should expire after duration")
	}
	// Touch 换目标时清零 lease
	s.MarkLease("k1")
	s.Touch("k1", "c2", false)
	if !s.LeaseExpired("k1") {
		t.Fatalf("Touch with new target should clear lease")
	}
}

func TestAffinityMiss(t *testing.T) {
	s := newTestAffinity(time.Minute, time.Second, 1000)
	defer s.Close()

	s.Touch("k1", "c1", false)
	if v := s.RecordMiss("k1"); v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
	if v := s.RecordMiss("k1"); v != 2 {
		t.Fatalf("expected 2, got %d", v)
	}
	// Touch 换目标时清零 MissCount
	s.Touch("k1", "c2", false)
	if v := s.RecordMiss("k1"); v != 1 {
		t.Fatalf("expected reset to 1, got %d", v)
	}
}

func TestAffinityLRU(t *testing.T) {
	// maxEntries=64 → 2/片
	s := newTestAffinity(time.Minute, time.Second, 64)
	defer s.Close()
	for i := 0; i < 128; i++ {
		s.Touch(string(rune('a'+i%26))+string(rune('0'+i/26)), "c", false)
	}
	if n := s.Len(); n > 64 {
		t.Fatalf("expected Len<=64, got %d", n)
	}
}

func TestAffinityConcurrent(t *testing.T) {
	s := newTestAffinity(time.Minute, time.Second, 1000)
	defer s.Close()
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				key := string(rune('a'+g)) + string(rune('0'+i%10))
				s.Touch(key, "c", false)
				s.Get(key)
				if i%7 == 0 {
					s.MarkLease(key)
					s.LeaseExpired(key)
				}
				if i%11 == 0 {
					s.RecordMiss(key)
				}
				if i%13 == 0 {
					s.Delete(key)
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestAffinityJanitor(t *testing.T) {
	s := newTestAffinity(30*time.Millisecond, time.Second, 1000)
	// 手动启动 janitor 用短周期
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
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
						}
						e = prev
					}
					sh.mu.Unlock()
				}
			}
		}
	}()
	defer s.Close()

	s.Touch("k1", "c1", false)
	time.Sleep(80 * time.Millisecond)
	if n := s.Len(); n != 0 {
		t.Fatalf("expected janitor to clear expired entries, Len=%d", n)
	}
}

func TestAffinityStats(t *testing.T) {
	// maxEntries=32 → maxPerShard=1，写 2 条触发 LRU 逐出
	s := newTestAffinity(time.Minute, time.Second, affinityShards)
	defer s.Close()

	// 命中
	s.Touch("k1", "c1", false)
	if _, ok := s.Get("k1"); !ok {
		t.Fatal("expected hit")
	}
	// 回迁（换目标）
	s.Touch("k1", "c2", false)
	// 实测失效删除
	s.Delete("k1")
	// LRU 逐出：maxPerShard=1，写 64 条保证同片 ≥2 条触发逐出
	for i := 0; i < 64; i++ {
		s.Touch(fmt.Sprintf("k%d", i), "c1", false)
	}

	st := s.Stats()
	if st.Hits != 1 {
		t.Fatalf("expected 1 hit, got %d", st.Hits)
	}
	if st.Rebinds != 1 {
		t.Fatalf("expected 1 rebind, got %d", st.Rebinds)
	}
	if st.MissDeletes != 1 {
		t.Fatalf("expected 1 miss delete, got %d", st.MissDeletes)
	}
	if st.Evictions < 1 {
		t.Fatalf("expected >=1 eviction, got %d", st.Evictions)
	}
	// 读取即清零
	st2 := s.Stats()
	if st2.Hits != 0 || st2.Rebinds != 0 || st2.MissDeletes != 0 {
		t.Fatalf("expected Stats to reset counters, got %+v", st2)
	}
}
