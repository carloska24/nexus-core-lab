package network

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrIPPoolExhausted    = errors.New("ip pool exhausted")
	ErrIPNotAllocated     = errors.New("ip address is not allocated in pool")
	ErrInvalidIPFormat    = errors.New("invalid ip format for pool")
	ErrIPAlreadyAllocated = errors.New("ip address is already allocated in pool")
)

const (
	baseIPv4Net = "10.45.0.0/16"
	minHost     = uint32(2)     // 10.45.0.2 (reserva .0 para rede e .1 para gateway simulado)
	maxHost     = uint32(65534) // 10.45.255.254 (reserva 10.45.255.255 para broadcast simulado)
)

// IPPool gerencia a alocação lógica e reciclagem de endereços IP no bloco 10.45.0.0/16.
// É thread-safe e previne corrupção de pool por liberações indevidas ou duplicadas.
type IPPool struct {
	mu        sync.Mutex
	nextHost  uint32
	allocated map[string]struct{}
	recycled  []string
}

// NewIPPool cria uma nova instância de IPPool baseada em 10.45.0.0/16.
func NewIPPool() *IPPool {
	return &IPPool{
		nextHost:  minHost,
		allocated: make(map[string]struct{}),
		recycled:  make([]string, 0),
	}
}

// Allocate reserva e retorna um endereço IPv4 virtual disponível.
func (p *IPPool) Allocate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Prioriza endereços reciclados
	if len(p.recycled) > 0 {
		lastIdx := len(p.recycled) - 1
		ip := p.recycled[lastIdx]
		p.recycled = p.recycled[:lastIdx]

		p.allocated[ip] = struct{}{}
		return ip, nil
	}

	if p.nextHost > maxHost {
		return "", ErrIPPoolExhausted
	}

	ip := hostToIPv4String(p.nextHost)
	p.nextHost++

	p.allocated[ip] = struct{}{}
	return ip, nil
}

// Release devolve um endereço IPv4 ao pool para futuro reaproveitamento.
// Retorna ErrIPNotAllocated se o endereço não estiver alocado, prevenindo corrupção do pool.
func (p *IPPool) Release(ip string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.allocated[ip]; !ok {
		return ErrIPNotAllocated
	}

	delete(p.allocated, ip)
	p.recycled = append(p.recycled, ip)
	return nil
}

// IsAllocated verifica se um determinado IP está atualmente reservado.
func (p *IPPool) IsAllocated(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, ok := p.allocated[ip]
	return ok
}

// AllocatedCount retorna o número total de endereços alocados no momento.
func (p *IPPool) AllocatedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.allocated)
}

func hostToIPv4String(host uint32) string {
	b3 := byte(host >> 8)
	b4 := byte(host & 0xFF)
	return net.IPv4(10, 45, b3, b4).String()
}

// MarkAllocated reserva explicitamente um endereço IP existente (usado no warm-up pós-restart).
// Retorna erro se o IP for inválido, fora do bloco 10.45.0.0/16 ou se já estiver alocado.
func (p *IPPool) MarkAllocated(ip string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ErrInvalidIPFormat
	}
	ipv4 := parsed.To4()
	if ipv4 == nil || ipv4[0] != 10 || ipv4[1] != 45 {
		return ErrInvalidIPFormat
	}

	host := (uint32(ipv4[2]) << 8) | uint32(ipv4[3])
	if host < minHost || host > maxHost {
		return ErrInvalidIPFormat
	}

	if _, ok := p.allocated[ip]; ok {
		return ErrIPAlreadyAllocated
	}

	// Remove de recycled se presente
	for i, r := range p.recycled {
		if r == ip {
			p.recycled = append(p.recycled[:i], p.recycled[i+1:]...)
			break
		}
	}

	// Se o IP estiver à frente ou igual a nextHost, avança nextHost para host + 1
	if host >= p.nextHost {
		p.nextHost = host + 1
	}

	p.allocated[ip] = struct{}{}
	return nil
}
