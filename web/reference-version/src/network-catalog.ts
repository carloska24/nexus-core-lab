// Mirror of internal/network/cell.go; configuration, NOT an HTTP discovery result.
// x/y are positions on the approved illustration, NOT GPS or physical tower locations.
export const networkCells = [
  { id: 'CELL-SP-001', name: 'Campinas Centro', tech: 'LTE', x: 215, y: 248 },
  { id: 'CELL-SP-002', name: 'Campinas Barão Geraldo', tech: '5G', x: 426, y: 49 },
  { id: 'CELL-SP-003', name: 'Campinas Cambuí', tech: '5G', x: 606, y: 253 },
] as const;
