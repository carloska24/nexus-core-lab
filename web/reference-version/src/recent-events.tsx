import {createContext,useContext,useEffect,useState,type ReactNode} from 'react';
import {read} from './api';
import type {ReadState} from './monitoring';

export const eventTypes=['ATTACH','CELL_HANDOVER','DETACH','STALE_DISCONNECT'] as const;
export type RecentEvent={id:string;type:typeof eventTypes[number];timestamp:string;device_id:string;subscriber_id:string;session_id:string;cell_id:string;ip_address?:string;from_cell_id?:string;to_cell_id?:string;disconnect_reason?:string};
function valid(value:unknown):value is RecentEvent[]{
  return Array.isArray(value)&&value.length<=100&&new Set(value.map(e=>e?.id)).size===value.length&&value.every(e=>e&&
    ['id','device_id','subscriber_id','session_id','cell_id'].every(k=>typeof e[k]==='string'&&e[k].length>0)&&
    eventTypes.includes(e.type)&&typeof e.timestamp==='string'&&Number.isFinite(Date.parse(e.timestamp))&&
    ['ip_address','from_cell_id','to_cell_id','disconnect_reason'].every(k=>e[k]===undefined||typeof e[k]==='string'));
}
export const getRecentEvents=(signal:AbortSignal)=>read('/api/v1/events/recent',valid,signal);
const Context=createContext<ReadState<RecentEvent[]>>({status:'loading'});
export function RecentEventsProvider({children}:{children:ReactNode}){
  const [state,setState]=useState<ReadState<RecentEvent[]>>({status:'loading'});
  useEffect(()=>{
    let disposed=false,timer:ReturnType<typeof setTimeout>,controller:AbortController;
    async function poll(){
      controller=new AbortController();const deadline=setTimeout(()=>controller.abort(),8000);
      try{const data=await getRecentEvents(controller.signal);if(!disposed)setState({status:'success',data,receivedAt:Date.now()});}
      catch(error){if(!disposed)setState(previous=>({...previous,status:previous.data===undefined?'error':'stale',error:error instanceof Error?error.message:'Network error'}));}
      finally{clearTimeout(deadline);if(!disposed)timer=setTimeout(poll,5000);}
    }
    void poll();return()=>{disposed=true;clearTimeout(timer);controller?.abort();};
  },[]);
  return <Context.Provider value={state}>{children}</Context.Provider>;
}
export const useRecentEvents=()=>useContext(Context);
