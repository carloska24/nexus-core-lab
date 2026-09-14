package device

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestGlobalDeviceHTTPContract(t *testing.T) {
	mux, checker := setupTestDeviceMux()
	checker.subscribers["sub-other"] = "ACTIVE"
	list := func(path string) []Device {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 {
			t.Fatalf("GET %s: %d %s", path, w.Code, w.Body.String())
		}
		var devices []Device
		if err := json.Unmarshal(w.Body.Bytes(), &devices); err != nil {
			t.Fatal(err)
		}
		return devices
	}
	if got := list("/api/v1/devices"); got == nil || len(got) != 0 {
		t.Fatalf("expected [], got %+v", got)
	}
	created := make([]Device, 0, 2)
	for i, sub := range []string{"sub-active", "sub-other"} {
		payload := fmt.Sprintf(`{"subscriber_id":%q,"imei":"4901542032375%02d","technology":"LTE"}`, sub, i)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewBufferString(payload)))
		if w.Code != 201 {
			t.Fatalf("register: %d %s", w.Code, w.Body.String())
		}
		var d Device
		if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
			t.Fatal(err)
		}
		created = append(created, d)
	}
	// The global contract orders by CreatedAt, then ID when timestamps tie.
	sort.Slice(created, func(i, j int) bool {
		if created[i].CreatedAt.Equal(created[j].CreatedAt) {
			return created[i].ID < created[j].ID
		}
		return created[i].CreatedAt.Before(created[j].CreatedAt)
	})
	all := list("/api/v1/devices")
	if len(all) != 2 || all[0].ID != created[0].ID || all[1].ID != created[1].ID {
		t.Fatalf("global: %+v", all)
	}
	for _, d := range created {
		filtered := list("/api/v1/devices?subscriber_id=" + d.SubscriberID)
		if len(filtered) != 1 || filtered[0].ID != d.ID {
			t.Fatalf("filter: %+v", filtered)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+d.ID, nil))
		var got Device
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || got != d {
			t.Fatalf("detail: %d %+v", w.Code, got)
		}
	}
	if got := list("/api/v1/devices?subscriber_id=unknown"); len(got) != 0 {
		t.Fatalf("empty filter: %+v", got)
	}
	body, err := json.Marshal(RegisterRequest{SubscriberID: created[0].SubscriberID, IMEI: created[0].IMEI, Technology: Tech5G})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body)))
	if w.Code != 409 {
		t.Fatalf("duplicate: %d %s", w.Code, w.Body.String())
	}
	if got := list("/api/v1/devices"); len(got) != 2 {
		t.Fatalf("duplicate changed collection: %+v", got)
	}
}

func TestMemoryDeviceGlobalListCopiesAndOrder(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	for _, id := range []string{"b", "a", "c"} {
		d := &Device{ID: id, IMEI: id, SubscriberID: id, CreatedAt: now}
		if id == "c" {
			d.CreatedAt = now.Add(-time.Second)
		}
		if err := repo.Save(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != "c" || got[1].ID != "a" || got[2].ID != "b" {
		t.Fatalf("order: %+v", got)
	}
	got[0].IMEI = "mutated"
	got[1] = nil
	next, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if next[0].IMEI != "c" || next[1] == nil {
		t.Fatal("list exposed internal data")
	}
}

func TestMemoryDeviceGlobalListConcurrent(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				id := fmt.Sprintf("%d-%d", worker, i)
				if err := repo.Save(ctx, &Device{ID: id, IMEI: id, SubscriberID: "sub"}); err != nil {
					t.Error(err)
					return
				}
				list, err := repo.List(ctx)
				if err != nil {
					t.Error(err)
					return
				}
				for _, d := range list {
					d.Status = StatusInactive
				}
			}
		}(worker)
	}
	wg.Wait()
	list, err := repo.List(ctx)
	if err != nil || len(list) != 240 {
		t.Fatalf("list: %d %v", len(list), err)
	}
	for _, d := range list {
		if d.Status == StatusInactive {
			t.Fatal("concurrent list mutated repository")
		}
	}
}

type failedDeviceListRepository struct{ Repository }

func (failedDeviceListRepository) List(context.Context) ([]*Device, error) {
	return nil, errors.New("unavailable")
}
func TestGlobalDeviceListFailure(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(NewService(failedDeviceListRepository{}, nil)).RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))
	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("unexpected error: %+v", body)
	}
}
