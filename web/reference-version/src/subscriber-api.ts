import { ApiError, read } from './api';

export type SubscriberStatus = 'PENDING_ACTIVATION' | 'ACTIVE' | 'SUSPENDED' | 'DEACTIVATED';
export interface Subscriber {
  id: string;
  imsi: string;
  msisdn: string;
  status: SubscriberStatus;
  suspension_reason?: string;
  deactivation_reason?: string;
  created_at: string;
  updated_at: string;
}
export interface ProvisionSubscriberRequest { imsi: string; msisdn: string }
export const subscriberStatuses: SubscriberStatus[] = ['PENDING_ACTIVATION', 'ACTIVE', 'SUSPENDED', 'DEACTIVATED'];
export const validIMSI = (value: string) => /^\d{15}$/.test(value);
export const validMSISDN = (value: string) => /^\+?[1-9]\d{9,14}$/.test(value);

function isSubscriber(value: unknown): value is Subscriber {
  if (typeof value !== 'object' || value === null) return false;
  const sub = value as Record<string, unknown>;
  return ['id','imsi','msisdn','status','created_at','updated_at'].every(key => typeof sub[key] === 'string') &&
    subscriberStatuses.includes(sub.status as SubscriberStatus) &&
    Number.isFinite(Date.parse(sub.created_at as string)) && Number.isFinite(Date.parse(sub.updated_at as string)) &&
    ['suspension_reason','deactivation_reason'].every(key => sub[key] === undefined || typeof sub[key] === 'string');
}
const base = '/api/v1/subscribers';
// Bounded requests; AbortSignal supplied by reads also supports route/unmount cancellation.
async function subscriberRequest<T>(path: string, validate: (value: unknown) => value is T, signal?: AbortSignal, options?: RequestInit): Promise<T> {
  const controller = new AbortController();
  const cancel = () => controller.abort();
  if (signal?.aborted) cancel();
  signal?.addEventListener('abort', cancel, { once: true });
  const timeout = setTimeout(cancel, 10000);
  try { return await read(path, validate, controller.signal, options); }
  finally { clearTimeout(timeout); signal?.removeEventListener('abort', cancel); }
}
export async function getSubscribers(signal?: AbortSignal): Promise<Subscriber[]> {
  const result = await subscriberRequest(base, (value): value is Subscriber[] | null => value === null || Array.isArray(value) && value.every(isSubscriber), signal);
  // Only after HTTP success AND payload validation, never as an error fallback.
  return result ?? [];
}
export function getSubscriber(id: string, signal?: AbortSignal): Promise<Subscriber> {
  return subscriberRequest(`${base}/${encodeURIComponent(id)}`, isSubscriber, signal);
}
export function getSubscriberByIMSI(imsi: string, signal?: AbortSignal): Promise<Subscriber> {
  return subscriberRequest(`${base}?imsi=${encodeURIComponent(imsi)}`, isSubscriber, signal);
}
export function provisionSubscriber(body: ProvisionSubscriberRequest): Promise<Subscriber> {
  return subscriberRequest(base, isSubscriber, undefined, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
}
export function activateSubscriber(id: string): Promise<Subscriber> {
  return subscriberRequest(`${base}/${encodeURIComponent(id)}/activate`, isSubscriber, undefined, { method: 'POST' });
}
export function suspendSubscriber(id: string, reason?: string): Promise<Subscriber> {
  return subscriberRequest(`${base}/${encodeURIComponent(id)}/suspend`, isSubscriber, undefined, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reason }) });
}
export function deactivateSubscriber(id: string, reason?: string): Promise<Subscriber> {
  return subscriberRequest(`${base}/${encodeURIComponent(id)}/deactivate`, isSubscriber, undefined, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reason }) });
}
export function subscriberError(error: unknown): string {
  if (error instanceof ApiError) return `${error.status ? `HTTP ${error.status} · ` : ''}${error.code ? `${error.code} · ` : ''}${error.message}`;
  if (error instanceof Error && error.name === 'AbortError') return 'Request timed out or was cancelled. Refresh to confirm the current server state before retrying.';
  return 'Network error. The operation could not be confirmed; refresh before retrying.';
}
