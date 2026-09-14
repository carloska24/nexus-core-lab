import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import { useDevices } from './devices';
import { getActiveSession, sessionError, type Session } from './session-api';
import type { Device } from './device-api';

export const TOPOLOGY_CONCURRENCY = 4;
export const TOPOLOGY_INTERVAL = 5000;
export interface TopologyEntry {
  device: Device;
  session?: Session | null;
  status: 'connected' | 'disconnected' | 'unknown' | 'stale';
  error?: string;
  receivedAt?: number;
}
export interface TopologySnapshot {
  status: 'loading' | 'success' | 'partial' | 'stale' | 'error';
  entries: TopologyEntry[];
  receivedAt?: number;
  error?: string;
}
const initial: TopologySnapshot = { status: 'loading', entries: [] };
const TopologyContext = createContext<TopologySnapshot>(initial);

export function TopologyProvider({ enabled, children }: { enabled: boolean; children: ReactNode }) {
  const { collection } = useDevices();
  const devices = useRef(collection);
  devices.current = collection;
  const snapshot = useRef(initial);
  const [state, setState] = useState(initial);
  useEffect(() => {
    if (!enabled) return;
    let disposed = false;
    let timer: ReturnType<typeof setTimeout>;
    let controller: AbortController;
    const publish = (next: TopologySnapshot) => {
      if (!disposed) { snapshot.current = next; setState(next); }
    };
    // Returning to a topology route must not label an old observation as current.
    if (snapshot.current.receivedAt) publish({ ...snapshot.current, status: 'stale', entries: snapshot.current.entries.map(entry => ({ ...entry, status: entry.receivedAt ? 'stale' : 'unknown' })) });
    const round = async () => {
      controller = new AbortController();
      const source = devices.current;
      if (!source.data) {
        publish({ ...snapshot.current, status: source.status === 'loading' ? 'loading' : snapshot.current.receivedAt ? 'stale' : 'error', error: source.error });
        if (!disposed) timer = setTimeout(round, source.status === 'loading' ? 250 : TOPOLOGY_INTERVAL);
        return;
      }
      const current = [...source.data].sort((a, b) => a.id.localeCompare(b.id));
      const previous = new Map(snapshot.current.entries.map(entry => [entry.device.id, entry]));
      const entries: TopologyEntry[] = new Array(current.length);
      let cursor = 0;
      const worker = async () => {
        while (!disposed && !controller.signal.aborted) {
          const index = cursor++;
          if (index >= current.length) return;
          const device = current[index];
          try {
            const session = await getActiveSession(device.id, controller.signal);
            if (session && (session.device_id !== device.id || session.subscriber_id !== device.subscriber_id || session.status !== 'CONNECTED')) throw new Error('Session association does not match device');
            entries[index] = { device, session, status: session ? 'connected' : 'disconnected', receivedAt: Date.now() };
          } catch (error) {
            const old = previous.get(device.id);
            entries[index] = { device, session: old?.session, receivedAt: old?.receivedAt, status: old?.receivedAt ? 'stale' : 'unknown', error: sessionError(error) };
          }
        }
      };
      await Promise.all(Array.from({ length: Math.min(TOPOLOGY_CONCURRENCY, current.length) }, worker));
      if (disposed) return;
      // A changed Devices collection will be picked up next round. This round is atomic.
      const failed = entries.filter(entry => entry.error).length;
      const hasKnown = entries.some(entry => entry.receivedAt !== undefined);
      let status: TopologySnapshot['status'] = failed === 0 ? 'success' : failed < entries.length ? 'partial' : hasKnown ? 'stale' : 'error';
      if (source.status !== 'success') {
        status = 'stale';
        entries.forEach(entry => { entry.status = entry.receivedAt ? 'stale' : 'unknown'; });
      }
      publish({ status, entries, receivedAt: Date.now(), error: source.error });
      // Completion-based scheduling: no overlapping rounds, including slow/time-out responses.
      timer = setTimeout(round, TOPOLOGY_INTERVAL);
    };
    void round();
    return () => { disposed = true; clearTimeout(timer); controller?.abort(); };
  }, [enabled]);
  return <TopologyContext.Provider value={state}>{children}</TopologyContext.Provider>;
}
export const useTopology = () => useContext(TopologyContext);
export function sessionDuration(attachedAt: string, now = Date.now()) {
  const seconds = Math.max(0, Math.floor((now - Date.parse(attachedAt)) / 1000));
  return [Math.floor(seconds / 3600), Math.floor(seconds / 60) % 60, seconds % 60].map(n => String(n).padStart(2, '0')).join(':');
}
