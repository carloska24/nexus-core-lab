import {
  ActiveSessionRow,
  ConnectedDeviceNode,
  EventTypeDistribution,
  IPPoolUsageData,
  KpiCardProps,
  NetworkCell,
  RecentEventRow,
  SessionActivityPoint,
} from '../types';

export const mockKpis: KpiCardProps[] = [
  {
    id: 'subscribers',
    label: 'Total Subscribers',
    value: '128',
    change: { value: '+12%', positive: true },
    subtitle: 'Provisioned in system',
    iconName: 'users',
    accentColor: 'var(--color-blue)',
  },
  {
    id: 'devices',
    label: 'Total Devices',
    value: '103',
    change: { value: '+8%', positive: true },
    subtitle: 'Registered devices',
    iconName: 'smartphone',
    accentColor: 'var(--color-blue)',
  },
  {
    id: 'sessions',
    label: 'Active Sessions',
    value: '17',
    change: { value: '+21%', positive: true },
    subtitle: 'Currently connected',
    iconName: 'wifi',
    accentColor: 'var(--color-blue)',
  },
  {
    id: 'cells',
    label: 'Network Cells',
    value: '3',
    badgeText: 'Online',
    subtitle: 'LTE / 5G · Campinas',
    iconName: 'radio',
    accentColor: 'var(--color-blue)',
  },
  {
    id: 'events',
    label: 'Total Events',
    value: '842',
    change: { value: '+35%', positive: true },
    subtitle: 'ATTACH / HANDOVER / DETACH',
    iconName: 'file-text',
    accentColor: 'var(--color-5g)',
  },
];

export const mockCells: NetworkCell[] = [
  {
    id: 'CELL-SP-001',
    name: 'Campinas Centro',
    region: 'Campinas - SP',
    tech: 'LTE',
    x: 270,
    y: 310,
  },
  {
    id: 'CELL-SP-002',
    name: 'Campinas Barão Geraldo',
    region: 'Campinas - SP',
    tech: '5G',
    x: 410,
    y: 150,
  },
  {
    id: 'CELL-SP-003',
    name: 'Campinas Cambuí',
    region: 'Campinas - SP',
    tech: '5G',
    x: 550,
    y: 310,
  },
];

export const mockConnectedDevices: ConnectedDeviceNode[] = [
  { id: 'UE-01', deviceId: '860010001234567', cellId: 'CELL-SP-001', x: 235, y: 385 },
  { id: 'UE-02', deviceId: '860010001234568', cellId: 'CELL-SP-002', x: 375, y: 225 },
  { id: 'UE-05', deviceId: '860010001234571', cellId: 'CELL-SP-002', x: 485, y: 220 },
  { id: 'UE-03', deviceId: '860010001234569', cellId: 'CELL-SP-003', x: 510, y: 395 },
  { id: 'UE-04', deviceId: '860010001234570', cellId: 'CELL-SP-003', x: 590, y: 380 },
];

export const mockActiveSessions: ActiveSessionRow[] = [
  {
    id: 'sess-1',
    deviceId: '860010001234567',
    subscriber: '5511999990001',
    cell: 'SP-001',
    ipAddress: '10.45.0.8',
    duration: '00:12:34',
  },
  {
    id: 'sess-2',
    deviceId: '860010001234568',
    subscriber: '5511999990002',
    cell: 'SP-002',
    ipAddress: '10.45.0.14',
    duration: '00:08:21',
  },
  {
    id: 'sess-3',
    deviceId: '860010001234569',
    subscriber: '5511999990003',
    cell: 'SP-003',
    ipAddress: '10.45.0.22',
    duration: '00:05:17',
  },
  {
    id: 'sess-4',
    deviceId: '860010001234570',
    subscriber: '5511999990004',
    cell: 'SP-002',
    ipAddress: '10.45.0.31',
    duration: '00:03:45',
  },
  {
    id: 'sess-5',
    deviceId: '860010001234571',
    subscriber: '5511999990005',
    cell: 'SP-001',
    ipAddress: '10.45.0.33',
    duration: '00:02:11',
  },
];

export const mockRecentEvents: RecentEventRow[] = [
  {
    id: 'evt-1',
    time: '23:41:14',
    type: 'DETACH',
    device: 'UE-05',
    details: 'Session terminated',
  },
  {
    id: 'evt-2',
    time: '23:41:09',
    type: 'CELL_HANDOVER',
    device: 'UE-01',
    details: 'SP-001 → SP-003',
  },
  {
    id: 'evt-3',
    time: '23:41:05',
    type: 'ATTACH',
    device: 'UE-04',
    details: 'Attached to SP-002',
  },
  {
    id: 'evt-4',
    time: '23:41:02',
    type: 'ATTACH',
    device: 'UE-03',
    details: 'Attached to SP-003',
  },
  {
    id: 'evt-5',
    time: '23:40:58',
    type: 'CELL_HANDOVER',
    device: 'UE-02',
    details: 'SP-002 → SP-001',
  },
];

export const mockSessionActivity: SessionActivityPoint[] = [
  { time: '23:10', activeSessions: 10, detachedSessions: 2 },
  { time: '23:15', activeSessions: 18, detachedSessions: 4 },
  { time: '23:20', activeSessions: 25, detachedSessions: 5 },
  { time: '23:25', activeSessions: 28, detachedSessions: 5 },
  { time: '23:30', activeSessions: 27, detachedSessions: 6 },
  { time: '23:35', activeSessions: 32, detachedSessions: 6 },
  { time: '23:40', activeSessions: 30, detachedSessions: 5 },
];

export const mockEventsByType: EventTypeDistribution[] = [
  { name: 'ATTACH', type: 'ATTACH', count: 354, percent: 42, color: '#10b981' },
  { name: 'CELL_HANDOVER', type: 'CELL_HANDOVER', count: 236, percent: 28, color: '#f43f5e' },
  { name: 'DETACH', type: 'DETACH', count: 202, percent: 24, color: '#f59e0b' },
  { name: 'OTHER', type: 'OTHER', count: 50, percent: 6, color: '#60a5fa' },
];

export const mockIPPoolUsage: IPPoolUsageData = {
  subnet: '10.45.0.0/16',
  typeLabel: '(Virtual Network)',
  used: 17,
  total: 256, // ilustrativo conforme instrução
  percent: 6.6,
  warmUpStatus: {
    completed: true,
    title: 'IPPool warm-up completed',
    subtitle: '1 active session IP(s) pre-allocated',
  },
};
