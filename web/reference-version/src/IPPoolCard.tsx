import {Check,Clock3,TriangleAlert,Server} from 'lucide-react';
import {usePolling,readLabel} from './monitoring';
import {getIPPool} from './ip-pool-api';
import './ip-pool.css';

export function IPPoolCard(){
  const state=usePolling(getIPPool,5000),pool=state.data;
  const count=(n:number|undefined)=>n===undefined?'—':n.toLocaleString('en-US');
  const percent=pool?`${pool.utilization_percent.toLocaleString('en-US',{maximumFractionDigits:4})}%`:'—';
  const StatusIcon=state.status==='success'?Check:state.status==='loading'?Clock3:TriangleAlert;
  return <article className="panel analytic pool live-ip-pool" data-testid="ip-pool" data-state={state.status}>
    <div className="panel-head"><div className="head-title"><span className="head-icon"><Server/></span><div><h2>IP Pool Usage</h2><p>{pool?`${pool.cidr} (Virtual Network)`:'Virtual network · awaiting pool snapshot'}</p></div></div></div>
    <div className="pool-stat"><b><span data-testid="pool-allocated">{count(pool?.allocated)}</span> / {count(pool?.capacity)} IPs allocated</b><b data-testid="pool-percent">{percent}</b></div>
    <div className="progress" role="progressbar" aria-label="IP pool utilization" aria-valuemin={0} aria-valuemax={100} aria-valuenow={pool?.utilization_percent}><i style={{width:pool?`${pool.utilization_percent}%`:'0%'}}/></div>
    <div className="pool-legend"><span><i/>Allocated ({count(pool?.allocated)})</span><span><i/>Available (<span data-testid="pool-available">{count(pool?.available)}</span>)</span></div>
    <div className="warm" role="status"><StatusIcon/><div><b>{readLabel(state)} · IP pool</b><small title={state.error}>{state.error|| (state.receivedAt?`Observed ${new Date(state.receivedAt).toLocaleTimeString('en-GB')} · allocator snapshot`:'Waiting for authoritative pool data')}</small></div></div>
  </article>;
}
