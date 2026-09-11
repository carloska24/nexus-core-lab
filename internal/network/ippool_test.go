package network

import (
	"errors"
	"sync"
	"testing"
)

func TestCellCatalog(t *testing.T) {
	cells := ListCells()
	if len(cells) != 3 {
		t.Fatalf("expected 3 static cells, got %d", len(cells))
	}

	cell, err := FindCell("CELL-SP-001")
	if err != nil {
		t.Fatalf("expected to find CELL-SP-001, got error: %v", err)
	}
	if cell.Name != "Campinas Centro" {
		t.Errorf("expected Campinas Centro, got %s", cell.Name)
	}

	_, err = FindCell("CELL-UNKNOWN")
	if !errors.Is(err, ErrCellNotFound) {
		t.Fatalf("expected ErrCellNotFound, got %v", err)
	}
}

func TestIPPool_AllocateAndRelease(t *testing.T) {
	pool := NewIPPool()

	ip1, err := pool.Allocate()
	if err != nil {
		t.Fatalf("unexpected allocation error: %v", err)
	}
	if ip1 != "10.45.0.2" {
		t.Errorf("expected initial IP to be 10.45.0.2, got %s", ip1)
	}

	ip2, err := pool.Allocate()
	if err != nil {
		t.Fatalf("unexpected allocation error: %v", err)
	}
	if ip2 != "10.45.0.3" {
		t.Errorf("expected second IP to be 10.45.0.3, got %s", ip2)
	}

	if pool.AllocatedCount() != 2 {
		t.Errorf("expected 2 allocated IPs, got %d", pool.AllocatedCount())
	}

	// Libera ip1 e valida reciclagem
	if err := pool.Release(ip1); err != nil {
		t.Fatalf("unexpected error releasing ip1: %v", err)
	}
	if pool.IsAllocated(ip1) {
		t.Errorf("expected ip1 to not be allocated")
	}

	// Próxima alocação deve reaproveitar ip1
	reusedIP, err := pool.Allocate()
	if err != nil {
		t.Fatalf("unexpected allocation error: %v", err)
	}
	if reusedIP != ip1 {
		t.Errorf("expected recycled IP %s, got %s", ip1, reusedIP)
	}
}

func TestIPPool_PreventDoubleReleaseAndCorruption(t *testing.T) {
	pool := NewIPPool()

	ip, err := pool.Allocate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Primeiro release com sucesso
	if err := pool.Release(ip); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}

	// Segundo release do mesmo IP deve falhar e não corromper a fila de reciclados
	err = pool.Release(ip)
	if !errors.Is(err, ErrIPNotAllocated) {
		t.Fatalf("expected ErrIPNotAllocated on double release, got %v", err)
	}

	// Release de IP inexistente
	err = pool.Release("10.45.99.99")
	if !errors.Is(err, ErrIPNotAllocated) {
		t.Fatalf("expected ErrIPNotAllocated on unallocated IP, got %v", err)
	}

	// Garante que alocar novamente entrega o IP apenas uma vez
	allocatedIP, err := pool.Allocate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allocatedIP != ip {
		t.Errorf("expected %s, got %s", ip, allocatedIP)
	}

	// A próxima alocação deve avançar o contador normalmente, sem duplicar o IP reciclado
	nextIP, err := pool.Allocate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextIP == allocatedIP {
		t.Errorf("critical corruption: IP %s was allocated twice", allocatedIP)
	}
}

func TestIPPool_ConcurrentAllocationAndRelease(t *testing.T) {
	pool := NewIPPool()
	var wg sync.WaitGroup
	workers := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ip, err := pool.Allocate()
			if err != nil {
				t.Errorf("allocation failed: %v", err)
				return
			}
			if err := pool.Release(ip); err != nil {
				t.Errorf("release failed: %v", err)
			}
		}()
	}

	wg.Wait()

	if pool.AllocatedCount() != 0 {
		t.Errorf("expected 0 allocated IPs at the end, got %d", pool.AllocatedCount())
	}
}

func TestIPPool_MarkAllocated(t *testing.T) {
	pool := NewIPPool()

	// Marca um IP válido
	if err := pool.MarkAllocated("10.45.0.10"); err != nil {
		t.Fatalf("expected MarkAllocated to succeed, got %v", err)
	}

	if !pool.IsAllocated("10.45.0.10") {
		t.Errorf("expected 10.45.0.10 to be allocated")
	}

	// Tentar marcar novamente deve retornar ErrIPAlreadyAllocated
	if err := pool.MarkAllocated("10.45.0.10"); !errors.Is(err, ErrIPAlreadyAllocated) {
		t.Errorf("expected ErrIPAlreadyAllocated, got %v", err)
	}

	// Tentar marcar IP inválido ou fora do bloco 10.45
	if err := pool.MarkAllocated("192.168.1.1"); !errors.Is(err, ErrInvalidIPFormat) {
		t.Errorf("expected ErrInvalidIPFormat for 192.168.1.1, got %v", err)
	}
	if err := pool.MarkAllocated("invalid-ip"); !errors.Is(err, ErrInvalidIPFormat) {
		t.Errorf("expected ErrInvalidIPFormat for invalid-ip, got %v", err)
	}
}
