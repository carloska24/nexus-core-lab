import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { getHealth, getTelemetry, type Health, type Telemetry } from './api';

export type ReadState<T> = {
  status: 'loading' | 'success' | 'error' | 'stale';
  data?: T;
  receivedAt?: number;
  error?: string;
};
export type Observation = { timestamp: string; active_sessions: number; gap: boolean };
const initial = { status: 'loading' } as const;
const Monitoring = createContext<{
  health: ReadState<Health>;
  telemetry: ReadState<Telemetry>;
  observations: Observation[];
}>({ health: initial, telemetry: initial, observations: [] });

// One subscription per resource for the entire app, not one per card/route.
// Schedule after completion so slow requests never overlap. All reads time out.
export function usePolling<T>(read: (signal: AbortSignal) => Promise<T>, interval: number, onSuccess?: (data: T, gap: boolean) => void) {
  const [state, setState] = useState<ReadState<T>>(initial);
  useEffect(() => {
    let disposed = false;
    let failed = false;
    let controller: AbortController;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      controller = new AbortController();
      const deadline = setTimeout(() => controller.abort(), 8000);
      try {
        const data = await read(controller.signal);
        if (!disposed) {
          setState({ status: 'success', data, receivedAt: Date.now() });
          onSuccess?.(data, failed);
          failed = false;
        }
      } catch (error) {
        failed = true;
        if (!disposed) setState(previous => ({ ...previous,
          status: previous.data === undefined ? 'error' : 'stale',
          error: error instanceof Error ? error.message : 'Network error',
        }));
      } finally {
        clearTimeout(deadline);
        if (!disposed) timer = setTimeout(poll, interval);
      }
    };
    void poll();
    return () => { disposed = true; clearTimeout(timer); controller?.abort(); };
  }, [read, interval, onSuccess]);
  return state;
}

export function MonitoringProvider({ children }: { children: ReactNode }) {
  const [observations, setObservations] = useState<Observation[]>([]);
  // Stable callback avoids resetting polling on every render.
  const [record] = useState(() => (data: Telemetry, gap: boolean) => {
    setObservations(previous => {
      const now = Date.parse(data.timestamp);
      const last = previous[previous.length - 1];
      if (last && now <= Date.parse(last.timestamp)) previous = [];
      return [...previous.filter(point => now - Date.parse(point.timestamp) <= 30 * 60 * 1000), {
        timestamp: data.timestamp, active_sessions: data.metrics.active_sessions,
        gap: gap || !!last && now - Date.parse(last.timestamp) > 15000,
      }].slice(-360);
    });
  });
  const health = usePolling(getHealth, 10000);
  const telemetry = usePolling(getTelemetry, 5000, record);
  return <Monitoring.Provider value={{ health, telemetry, observations }}>{children}</Monitoring.Provider>;
}

export const useMonitoring = () => useContext(Monitoring);
export function readLabel<T>(state: ReadState<T>) {
  if (state.status === 'success') return 'Live';
  if (state.status === 'stale') return 'Stale · last known';
  if (state.status === 'error') return 'Unavailable';
  return 'Loading…';
}
