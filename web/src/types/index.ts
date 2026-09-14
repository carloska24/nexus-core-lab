export type CellTech = 'LTE' | '5G';

export interface NetworkCell {
  id: string;
  name: string;
  region: string;
  tech: CellTech;
  x: number; // Porcentagem no SVG
  y: number; // Porcentagem no SVG
}

export interface ConnectedDeviceNode {
  id: string; // ex: UE-01
  deviceId: string; // IMEI truncado
  cellId: string;
  x: number;
  y: number;
}

export interface KpiCardProps {
  id: string;
  label: string;
  value: string | number;
  change?: {
    value: string;
    positive: boolean;
  };
  badgeText?: string;
  subtitle: string;
  iconName: 'users' | 'smartphone' | 'wifi' | 'radio' | 'file-text';
  accentColor?: string;
}

export interface ActiveSessionRow {
  id: string;
  deviceId: string;
  subscriber: string;
  cell: string;
  ipAddress: string;
  duration: string;
}

export type EventType = 'ATTACH' | 'CELL_HANDOVER' | 'DETACH' | 'STALE_DISCONNECT';

export interface RecentEventRow {
  id: string;
  time: string;
  type: EventType;
  device: string;
  details: string;
}

export interface SessionActivityPoint {
  time: string;
  activeSessions: number;
  detachedSessions: number;
}

export interface EventTypeDistribution {
  name: string;
  type: EventType | 'OTHER';
  count: number;
  percent: number;
  color: string;
}

export interface IPPoolUsageData {
  subnet: string;
  typeLabel: string;
  used: number;
  total: number;
  percent: number;
  warmUpStatus: {
    completed: boolean;
    title: string;
    subtitle: string;
  };
}
