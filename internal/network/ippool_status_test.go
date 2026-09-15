package network

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestIPPoolWarmupHighAddressLeavesHolesReachable(t *testing.T) {
	p := NewIPPool()
	if err := p.MarkAllocated("10.45.255.254"); err != nil {
		t.Fatal(err)
	}
	ip, err := p.Allocate()
	if err != nil {
		t.Fatalf("65532 usable holes remain after warm-up, Allocate failed: %v", err)
	}
	if ip == "10.45.255.254" {
		t.Fatal("allocated persisted address twice")
	}
}

func TestIPPoolSnapshotCapacityExhaustionAndWarmup(t *testing.T) {
	p := NewIPPool()
	initial := p.Snapshot()
	if initial.CIDR != "10.45.0.0/16" || initial.Capacity != 65533 || initial.Allocated != 0 || initial.Available != 65533 || initial.UtilizationPercent != 0 {
		t.Fatalf("initial: %+v", initial)
	}
	for _, ip := range []string{"10.45.0.0", "10.45.0.1", "10.45.255.255", "10.46.0.2", "invalid"} {
		if !errors.Is(p.MarkAllocated(ip), ErrInvalidIPFormat) {
			t.Fatalf("accepted reserved/invalid %s", ip)
		}
	}
	for _, ip := range []string{"10.45.255.254", "10.45.128.1", "10.45.0.10"} {
		if err := p.MarkAllocated(ip); err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(p.MarkAllocated("10.45.0.10"), ErrIPAlreadyAllocated) {
		t.Fatal("duplicate mark accepted")
	}
	if !errors.Is(p.MarkAllocated("::ffff:10.45.0.10"), ErrIPAlreadyAllocated) {
		t.Fatal("mapped IPv4 duplicated canonical host")
	}
	before := p.nextHost
	for i := 0; i < 10; i++ {
		_ = p.Snapshot()
	}
	if p.nextHost != before {
		t.Fatal("snapshot changed cursor")
	}
	seen := map[string]bool{"10.45.255.254": true, "10.45.128.1": true, "10.45.0.10": true}
	for i := 0; i < 65530; i++ {
		ip, err := p.Allocate()
		if err != nil {
			t.Fatalf("lost capacity at %d: %v", i, err)
		}
		if seen[ip] {
			t.Fatalf("duplicate: %s", ip)
		}
		seen[ip] = true
	}
	full := p.Snapshot()
	if full.Allocated != 65533 || full.Available != 0 || full.UtilizationPercent != 100 {
		t.Fatalf("full: %+v", full)
	}
	if _, err := p.Allocate(); !errors.Is(err, ErrIPPoolExhausted) {
		t.Fatalf("exhaustion: %v", err)
	}
	if err := p.Release("10.45.0.2"); err != nil {
		t.Fatal(err)
	}
	if p.Snapshot().Available != 1 {
		t.Fatal("release did not restore capacity")
	}
	ip, err := p.Allocate()
	if err != nil || ip != "10.45.0.2" {
		t.Fatalf("reuse: %s %v", ip, err)
	}
	if p.Snapshot() != full {
		t.Fatal("reuse changed capacity")
	}
}

func TestIPPoolMarkAllocatedRemovesRecycled(t *testing.T) {
	p := NewIPPool()
	ip, _ := p.Allocate()
	if err := p.Release(ip); err != nil {
		t.Fatal(err)
	}
	if err := p.MarkAllocated(ip); err != nil {
		t.Fatal(err)
	}
	next, err := p.Allocate()
	if err != nil || next == ip || p.Snapshot().Allocated != 2 {
		t.Fatal("recycled warm-up duplicate")
	}
}

func TestIPPoolSnapshotConcurrent(t *testing.T) {
	p := NewIPPool()
	var wg sync.WaitGroup
	ips := make(chan string, 800)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				ip, err := p.Allocate()
				if err != nil {
					t.Error(err)
					return
				}
				ips <- ip
				s := p.Snapshot()
				if s.Allocated+s.Available != s.Capacity || s.UtilizationPercent != float64(s.Allocated)/float64(s.Capacity)*100 {
					t.Errorf("inconsistent snapshot: %+v", s)
				}
			}
		}()
	}
	wg.Wait()
	close(ips)
	seen := map[string]bool{}
	for ip := range ips {
		if seen[ip] {
			t.Fatal("duplicate concurrent allocation")
		}
		seen[ip] = true
	}
	if len(seen) != 800 || p.Snapshot().Allocated != 800 {
		t.Fatal("concurrent count")
	}
	for ip := range seen {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			if err := p.Release(ip); err != nil {
				t.Error(err)
			}
			_ = p.Snapshot()
		}(ip)
	}
	wg.Wait()
	if p.Snapshot().Allocated != 0 {
		t.Fatal("release count")
	}
}

func TestIPPoolHTTPReadOnly(t *testing.T) {
	p := NewIPPool()
	mux := http.NewServeMux()
	NewIPPoolHandler(p).RegisterRoutes(mux)
	for _, allocated := range []int{0, 1} {
		if allocated == 1 {
			if _, err := p.Allocate(); err != nil {
				t.Fatal(err)
			}
		}
		before := p.Snapshot()
		cursor := p.nextHost
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/network/ip-pool", nil))
			if w.Code != 200 || w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("HTTP contract")
			}
			var s IPPoolSnapshot
			if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
				t.Fatal(err)
			}
			if s != before {
				t.Fatalf("JSON: %+v", s)
			}
		}
		if p.nextHost != cursor || p.Snapshot() != before {
			t.Fatal("HTTP read mutated pool")
		}
	}
	for _, method := range []string{"POST", "DELETE", "PUT"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/network/ip-pool", nil))
		if w.Code != 405 {
			t.Fatalf("mutation route accepted: %s", method)
		}
	}
}
