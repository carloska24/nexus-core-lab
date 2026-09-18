import { networkCells } from './network-catalog.ts';
import type { TopologyEntry } from './topology.tsx';

export type Coordinate = [number, number];

export interface PresentationDevice {
  id: string;
  alias: string;
  imei: string;
  technology: string;
  cellId: string;
  cellName: string;
  sessionId: string;
  ipAddress: string;
  state: 'connected' | 'stale';
  coordinate: Coordinate;
}

export interface MapPresentation {
  cells: GeoJSON.FeatureCollection<GeoJSON.Point>;
  devices: GeoJSON.FeatureCollection<GeoJSON.Point>;
  links: GeoJSON.FeatureCollection<GeoJSON.LineString>;
  connected: PresentationDevice[];
}

function stableHash(value: string) {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

export function presentationCoordinate(key: string, cellId: string): Coordinate {
  const cell = networkCells.find(item => item.id === cellId);
  if (!cell) throw new Error(`Unknown configured cell: ${cellId}`);
  const hash = stableHash(key);
  const angle = (hash % 360) * Math.PI / 180;
  const radius = 0.008 + ((hash >>> 9) % 1000) / 1000 * 0.004;
  const longitudeScale = Math.max(0.4, Math.cos(cell.latitude * Math.PI / 180));
  return [
    Number((cell.longitude + Math.cos(angle) * radius / longitudeScale).toFixed(6)),
    Number((cell.latitude + Math.sin(angle) * radius).toFixed(6)),
  ];
}

export function buildMapPresentation(entries: TopologyEntry[]): MapPresentation {
  const connected = entries
    .filter((entry): entry is TopologyEntry & { session: NonNullable<TopologyEntry['session']>; status: 'connected' | 'stale' } =>
      Boolean(entry.session?.status === 'CONNECTED' && (entry.status === 'connected' || entry.status === 'stale')))
    .filter(entry => networkCells.some(cell => cell.id === entry.session.cell_id))
    .sort((left, right) => left.device.id.localeCompare(right.device.id))
    .map(entry => {
      const cell = networkCells.find(item => item.id === entry.session.cell_id)!;
      return {
        id: entry.device.id,
        alias: `D-${entry.device.imei.slice(-4)}`,
        imei: entry.device.imei,
        technology: entry.device.technology,
        cellId: cell.id,
        cellName: cell.name,
        sessionId: entry.session.id,
        ipAddress: entry.session.ip_address,
        state: entry.status,
        coordinate: presentationCoordinate(`${entry.session.id}:${entry.device.id}`, cell.id),
      } satisfies PresentationDevice;
    });

  const sessionCounts = new Map(networkCells.map(cell => [cell.id, connected.filter(device => device.cellId === cell.id).length]));
  const cells: GeoJSON.FeatureCollection<GeoJSON.Point> = {
    type: 'FeatureCollection',
    features: networkCells.map(cell => {
      const sessionCount = sessionCounts.get(cell.id) ?? 0;
      return {
        type: 'Feature',
        id: cell.id,
        geometry: { type: 'Point', coordinates: [cell.longitude, cell.latitude] },
        properties: {
          id: cell.id, name: cell.name, region: cell.region, tech: cell.tech,
          status: 'CONFIGURED', sessionCount, activity: sessionCount > 0 ? 'ACTIVE' : 'IDLE',
        },
      };
    }),
  };
  const devices: GeoJSON.FeatureCollection<GeoJSON.Point> = {
    type: 'FeatureCollection',
    features: connected.map(device => ({
      type: 'Feature',
      id: device.id,
      geometry: { type: 'Point', coordinates: device.coordinate },
      properties: { ...device, coordinate: undefined },
    })),
  };
  const links: GeoJSON.FeatureCollection<GeoJSON.LineString> = {
    type: 'FeatureCollection',
    features: connected.map(device => {
      const cell = networkCells.find(item => item.id === device.cellId)!;
      return {
        type: 'Feature',
        id: device.sessionId,
        geometry: { type: 'LineString', coordinates: [device.coordinate, [cell.longitude, cell.latitude]] },
        properties: { deviceId: device.id, sessionId: device.sessionId, cellId: device.cellId, state: device.state },
      };
    }),
  };
  return { cells, devices, links, connected };
}

export interface MapCapability {
  webgl: boolean;
  initializationFailed?: boolean;
  providerFailed?: boolean;
}

export function mapMode(capability: MapCapability): 'interactive' | 'fallback' {
  return capability.webgl && !capability.initializationFailed && !capability.providerFailed ? 'interactive' : 'fallback';
}
