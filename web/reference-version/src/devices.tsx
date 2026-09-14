import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import type { ReadState } from './monitoring';
import { getDevices, type Device } from './device-api';
import { deviceError } from './device-api';

const Devices = createContext<{
  collection: ReadState<Device[]>;
  refreshing: boolean;
  refresh: () => Promise<void>;
  accept: (device: Device) => void;
}>({ collection: {status:'loading'}, refreshing:false, refresh:async()=>{}, accept:()=>{} });

export function DevicesProvider({children}: {children:ReactNode}) {
  const [collection, setCollection] = useState<ReadState<Device[]>>({status:'loading'});
  const [refreshing, setRefreshing] = useState(false);
  const request = useRef<AbortController>();
  const generation = useRef(0);
  const mounted = useRef(false);
  const refresh = useCallback(async () => {
    if (!mounted.current) return;
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    const current = ++generation.current;
    setRefreshing(true);
    setCollection(previous => previous.data === undefined ? {status:'loading'} : previous);
    try {
      const data = await getDevices(controller.signal);
      if (mounted.current && current === generation.current) setCollection({status:'success',data,receivedAt:Date.now()});
    } catch (error) {
      if (mounted.current && current === generation.current) setCollection(previous => ({...previous,status:previous.data === undefined?'error':'stale',error:deviceError(error)}));
    } finally {
      if (mounted.current && current === generation.current) setRefreshing(false);
    }
  }, []);
  const accept = useCallback((device: Device) => {
    if (!mounted.current) return;
    // Cancel any older list snapshot before applying a confirmed server entity.
    request.current?.abort();
    ++generation.current;
    setCollection(previous => previous.data === undefined ? previous : {
      ...previous,
      data: previous.data.some(item => item.id === device.id)
        ? previous.data.map(item => item.id === device.id ? device : item)
        : [...previous.data, device],
    });
    // A complete collection is still required, especially after initial load failure.
    void refresh();
  }, [refresh]);
  useEffect(() => {
    mounted.current = true;
    void refresh();
    return () => { mounted.current = false; ++generation.current; request.current?.abort(); };
  }, [refresh]);
  return <Devices.Provider value={{collection,refreshing,refresh,accept}}>{children}</Devices.Provider>;
}
export const useDevices = () => useContext(Devices);

