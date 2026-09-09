package network

import (
	"errors"
	"sync"
)

var (
	ErrCellNotFound = errors.New("cell not found")
)

// Cell representa a entidade semântica de uma célula (antena/estação rádio base) simulada.
type Cell struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Tech   string `json:"tech"`
}

// Catálogo estático em memória representando a topologia de telecomunicações de Campinas/SP.
var staticCells = []Cell{
	{
		ID:     "CELL-SP-001",
		Name:   "Campinas Centro",
		Region: "Campinas - SP",
		Tech:   "LTE",
	},
	{
		ID:     "CELL-SP-002",
		Name:   "Campinas Barão Geraldo",
		Region: "Campinas - SP",
		Tech:   "5G",
	},
	{
		ID:     "CELL-SP-003",
		Name:   "Campinas Cambuí",
		Region: "Campinas - SP",
		Tech:   "5G",
	},
}

var cellsByID = sync.OnceValue(func() map[string]Cell {
	m := make(map[string]Cell, len(staticCells))
	for _, c := range staticCells {
		m[c.ID] = c
	}
	return m
})

// FindCell busca uma célula pelo seu identificador estático.
func FindCell(id string) (*Cell, error) {
	c, ok := cellsByID()[id]
	if !ok {
		return nil, ErrCellNotFound
	}
	return &c, nil
}

// ListCells retorna todas as células configuradas na topologia estática.
func ListCells() []Cell {
	result := make([]Cell, len(staticCells))
	copy(result, staticCells)
	return result
}
