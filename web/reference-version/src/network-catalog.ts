// Mirror of internal/network/cell.go; configuration, NOT an HTTP discovery result.
// x/y support the local fallback. longitude/latitude are illustrative presentation
// anchors for the named Campinas regions, never physical telecom infrastructure.
export const networkCells = [
  { id: 'CELL-SP-001', name: 'Campinas Centro', region: 'Centro', tech: 'LTE', x: 215, y: 248, longitude: -47.0608, latitude: -22.9056 },
  { id: 'CELL-SP-002', name: 'Campinas Barão Geraldo', region: 'Barão Geraldo', tech: '5G', x: 426, y: 49, longitude: -47.0714, latitude: -22.8238 },
  { id: 'CELL-SP-003', name: 'Campinas Cambuí', region: 'Cambuí', tech: '5G', x: 606, y: 253, longitude: -47.0508, latitude: -22.8970 },
] as const;
