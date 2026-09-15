import {read} from './api';
import type {ReadState} from './monitoring';
export type StorageSnapshot = {mode:'MEMORY';database_configured:false;database_status:'NOT_APPLICABLE'} | {mode:'POSTGRESQL';database_configured:true;database_status:'AVAILABLE'|'UNAVAILABLE'};
function valid(value:unknown):value is StorageSnapshot {
 if(!value||typeof value!=='object')return false;
 const s=value as StorageSnapshot;
 return s.mode==='MEMORY'?s.database_configured===false&&s.database_status==='NOT_APPLICABLE':s.mode==='POSTGRESQL'&&s.database_configured===true&&['AVAILABLE','UNAVAILABLE'].includes(s.database_status);
}
export const getStorage=(signal:AbortSignal)=>read('/api/v1/system/storage',valid,signal);
export function storageView(state:ReadState<StorageSnapshot>){
 const known=state.data?.mode==='MEMORY'?'MEMORY':state.data?`POSTGRESQL · ${state.data.database_status==='AVAILABLE'?'ONLINE':'OFFLINE'}`:'';
 if(state.status==='loading')return {label:'Checking…',dot:'loading'};
 if(state.status==='error')return {label:'Unknown',dot:'unknown'};
 if(state.status==='stale')return {label:`Stale · ${known}`,dot:'unknown'};
 return {label:known,dot:state.data?.database_status==='UNAVAILABLE'?'offline':'online'};
}
