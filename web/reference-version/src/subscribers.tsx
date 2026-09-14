import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import type { ReadState } from './monitoring';
import { getSubscribers, subscriberError, type Subscriber } from './subscriber-api';

const Subscribers = createContext<{
  collection: ReadState<Subscriber[]>;
  refreshing: boolean;
  refresh: () => Promise<void>;
  accept: (subscriber: Subscriber) => void;
}>({ collection: {status:'loading'}, refreshing:false, refresh:async()=>{}, accept:()=>{} });

export function SubscribersProvider({children}: {children:ReactNode}) {
  const [collection, setCollection] = useState<ReadState<Subscriber[]>>({status:'loading'});
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
      const data = await getSubscribers(controller.signal);
      if (mounted.current && current === generation.current) setCollection({status:'success',data,receivedAt:Date.now()});
    } catch (error) {
      if (mounted.current && current === generation.current) setCollection(previous => ({...previous,status:previous.data === undefined?'error':'stale',error:subscriberError(error)}));
    } finally {
      if (mounted.current && current === generation.current) setRefreshing(false);
    }
  }, []);
  const accept = useCallback((subscriber: Subscriber) => {
    if (!mounted.current) return;
    // Cancel any older list snapshot before applying a confirmed server entity.
    request.current?.abort();
    ++generation.current;
    setCollection(previous => previous.data === undefined ? previous : {
      ...previous,
      data: previous.data.some(item => item.id === subscriber.id)
        ? previous.data.map(item => item.id === subscriber.id ? subscriber : item)
        : [...previous.data, subscriber],
    });
    // A complete collection is still required, especially after initial load failure.
    void refresh();
  }, [refresh]);
  useEffect(() => {
    mounted.current = true;
    void refresh();
    return () => { mounted.current = false; ++generation.current; request.current?.abort(); };
  }, [refresh]);
  return <Subscribers.Provider value={{collection,refreshing,refresh,accept}}>{children}</Subscribers.Provider>;
}
export const useSubscribers = () => useContext(Subscribers);
