import { ApiError, read } from './api';

export type DeviceTechnology = 'LTE' | '5G';
export type DeviceStatus = 'REGISTERED' | 'INACTIVE';
export interface Device {
  id: string;
  subscriber_id: string;
  imei: string;
  technology: DeviceTechnology;
  status: DeviceStatus;
  created_at: string;
  updated_at: string;
}
export interface RegisterDeviceRequest { subscriber_id: string; imei: string; technology: DeviceTechnology }
export const validIMEI = (value: string) => /^\d{15}$/.test(value);

function isDevice(value: unknown): value is Device {
  if (typeof value !== 'object' || value === null) return false;
  const d = value as Record<string, unknown>;
  return ['id', 'subscriber_id', 'imei', 'created_at', 'updated_at'].every(key => typeof d[key] === 'string') &&
    ['LTE', '5G'].includes(d.technology as string) && ['REGISTERED', 'INACTIVE'].includes(d.status as string) &&
    Number.isFinite(Date.parse(d.created_at as string)) && Number.isFinite(Date.parse(d.updated_at as string));
}
async function request<T>(path: string, validate: (value: unknown) => value is T, signal?: AbortSignal, options?: RequestInit): Promise<T> {
  const controller = new AbortController();
  const cancel = () => controller.abort();
  if (signal?.aborted) cancel();
  signal?.addEventListener('abort', cancel, { once: true });
  const timer = setTimeout(cancel, 10000);
  try { return await read(path, validate, controller.signal, options); }
  finally { clearTimeout(timer); signal?.removeEventListener('abort', cancel); }
}
const base = '/api/v1/devices';
async function list(path: string, signal?: AbortSignal): Promise<Device[]> {
  const data = await request(path, (value): value is Device[] | null => value === null || Array.isArray(value) && value.every(isDevice), signal);
  // Normalize only a successful and validated null collection, never a failed request.
  return data ?? [];
}
export const getDevices = (signal?: AbortSignal) => list(base, signal);
export const getDevice = (id: string, signal?: AbortSignal) => request(`${base}/${encodeURIComponent(id)}`, isDevice, signal);
export const getDevicesBySubscriber = (id: string, signal?: AbortSignal) => list(`${base}?subscriber_id=${encodeURIComponent(id)}`, signal);
export const registerDevice = (body: RegisterDeviceRequest) => request(base, isDevice, undefined, {
  method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
});
export function deviceError(error: unknown): string {
  if (error instanceof ApiError) return `${error.status ? `HTTP ${error.status} · ` : ''}${error.code ? `${error.code} · ` : ''}${error.message}`;
  if (error instanceof Error && error.name === 'AbortError') return 'Request timed out or was cancelled. Refresh before retrying.';
  return 'Network error. The operation could not be confirmed; refresh before retrying.';
}
