import {read} from './api';

export type IPPoolSnapshot={cidr:string;capacity:number;allocated:number;available:number;utilization_percent:number};
function valid(value:unknown):value is IPPoolSnapshot {
  if(!value||typeof value!=='object')return false;
  const p=value as IPPoolSnapshot;
  return typeof p.cidr==='string'&&p.cidr.length>0&&Number.isSafeInteger(p.capacity)&&p.capacity>0&&
    Number.isSafeInteger(p.allocated)&&p.allocated>=0&&Number.isSafeInteger(p.available)&&p.available>=0&&
    p.allocated+p.available===p.capacity&&typeof p.utilization_percent==='number'&&Number.isFinite(p.utilization_percent)&&
    p.utilization_percent>=0&&p.utilization_percent<=100&&Math.abs(p.utilization_percent-p.allocated/p.capacity*100)<1e-8;
}
export const getIPPool=(signal:AbortSignal)=>read('/api/v1/network/ip-pool',valid,signal);
