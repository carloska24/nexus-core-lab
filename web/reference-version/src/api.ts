export interface Health {
  status: string;
  service: string;
}

export interface Telemetry {
  service: string;
  timestamp: string;
  requests_total: number;
  metrics: {
    active_sessions: number;
    connected_devices: number;
    dropped_events_total: number;
  };
  events_total: {
    attach: number;
    cell_handover: number;
    detach: number;
    stale_disconnect: number;
  };
}

export class ApiError extends Error {
  constructor(message: string, public status?: number, public code?: string) {
    super(message);
    this.name = 'ApiError';
  }
}

const object = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;
const count = (value: unknown) => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;

// Runtime validation prevents an HTML proxy response or malformed JSON from
// becoming an ONLINE status or a graph of invented zeroes.
function isHealth(value: unknown): value is Health {
  return object(value) && typeof value.status === 'string' && typeof value.service === 'string';
}
function isTelemetry(value: unknown): value is Telemetry {
  if (!object(value) || typeof value.service !== 'string' || typeof value.timestamp !== 'string' ||
      !Number.isFinite(Date.parse(value.timestamp)) || !count(value.requests_total) ||
      !object(value.metrics) || !object(value.events_total)) return false;
  return ['active_sessions', 'connected_devices', 'dropped_events_total'].every(key => count(value.metrics && (value.metrics as Record<string, unknown>)[key])) &&
    ['attach', 'cell_handover', 'detach', 'stale_disconnect'].every(key => count((value.events_total as Record<string, unknown>)[key]));
}

export async function read<T>(path: string, validate: (value: unknown) => value is T, signal?: AbortSignal, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...options, signal, cache: 'no-store', headers: { Accept: 'application/json', ...options.headers } });
  if (!response.ok) {
    let detail: unknown;
    try { detail = await response.json(); } catch { /* Proxy failures may have no JSON body. */ }
    throw new ApiError(object(detail) && typeof detail.message === 'string' ? detail.message : `HTTP ${response.status}`, response.status,
      object(detail) && typeof detail.code === 'string' ? detail.code : undefined);
  }
  const data: unknown = await response.json();
  if (!validate(data)) throw new ApiError(`Invalid response from ${path}`);
  return data;
}

export async function getHealth(signal?: AbortSignal): Promise<Health> {
  const health = await read('/health', isHealth, signal);
  if (health.status !== 'ok' || health.service !== 'nexus-core-lab') throw new ApiError('Unexpected health status');
  return health;
}
export function getTelemetry(signal?: AbortSignal): Promise<Telemetry> {
  return read('/telemetry', isTelemetry, signal);
}

export function totalEvents(telemetry: Telemetry): number {
  const events = telemetry.events_total;
  return events.attach + events.cell_handover + events.detach + events.stale_disconnect;
}
