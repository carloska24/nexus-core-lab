import { ApiError, read } from './api';

export interface Session {
  id: string; device_id: string; subscriber_id: string; cell_id: string; ip_address: string;
  status: 'CONNECTED' | 'DISCONNECTED'; attached_at: string; updated_at: string;
  closed_at?: string; disconnect_reason?: string;
}
// Compatibility export for Sessions; one shared catalogue for the frontend.
export { networkCells as sessionCells } from './network-catalog';
function valid(value: unknown): value is Session {
  if (!value || typeof value !== 'object') return false;
  const s = value as Record<string, unknown>;
  return ['id','device_id','subscriber_id','cell_id','ip_address'].every(k => typeof s[k] === 'string') &&
    ['CONNECTED','DISCONNECTED'].includes(s.status as string) &&
    ['attached_at','updated_at'].every(k => typeof s[k] === 'string' && Number.isFinite(Date.parse(s[k] as string))) &&
    (s.closed_at === undefined || typeof s.closed_at === 'string' && Number.isFinite(Date.parse(s.closed_at))) &&
    (s.disconnect_reason === undefined || typeof s.disconnect_reason === 'string');
}
async function request(path: string, signal?: AbortSignal, body?: object) {
  const controller = new AbortController();
  const cancel = () => controller.abort();
  if (signal?.aborted) cancel();
  signal?.addEventListener('abort', cancel, { once: true });
  const timeout = setTimeout(cancel, 10000);
  try { return await read(path, valid, controller.signal, body === undefined ? {} : {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
  }); } finally { clearTimeout(timeout); signal?.removeEventListener('abort', cancel); }
}
const base = '/api/v1/sessions';
export async function getActiveSession(deviceID: string, signal?: AbortSignal): Promise<Session | null> {
  try { return await request(`${base}?device_id=${encodeURIComponent(deviceID)}`, signal); }
  catch (error) {
    if (error instanceof ApiError && error.status === 404 && error.code === 'ACTIVE_SESSION_NOT_FOUND') return null;
    throw error;
  }
}
export const attachSession = (device_id: string, cell_id: string) => request(`${base}/attach`, undefined, { device_id, cell_id });
export const handoverSession = (id: string, target_cell_id: string) => request(`${base}/${encodeURIComponent(id)}/handover`, undefined, { target_cell_id });
export const detachSession = (id: string) => request(`${base}/${encodeURIComponent(id)}/detach`, undefined, {});
export function sessionError(error: unknown) {
  if (error instanceof ApiError) return `${error.status ? `HTTP ${error.status} · ` : ''}${error.code ? `${error.code} · ` : ''}${error.message}`;
  return 'Connection interrupted or request timed out. Refresh to confirm server state before retrying.';
}
