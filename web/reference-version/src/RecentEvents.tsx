import {useState} from 'react';
import {ArrowRight,FileText,X} from 'lucide-react';
import {useRecentEvents,type RecentEvent} from './recent-events';
import './recent-events.css';

const short=(id:string)=>id.slice(0,8);
function description(e:RecentEvent){
  if(e.type==='CELL_HANDOVER')return e.from_cell_id&&e.to_cell_id?`${e.from_cell_id.replace('CELL-','')} → ${e.to_cell_id.replace('CELL-','')}`:e.cell_id;
  return e.type==='ATTACH'?`${e.cell_id.replace('CELL-','')}${e.ip_address?' · '+e.ip_address:''}`:e.disconnect_reason||e.cell_id;
}
export function RecentEvents({full=false}:{full?:boolean}){
  const state=useRecentEvents(),[selected,setSelected]=useState<RecentEvent>();
  const rows=full?state.data:state.data?.slice(0,5);
  const label=state.status==='success'?'Live':state.status==='stale'?'Stale · last known':state.status==='error'?'Unavailable':'Loading…';
  const panel=<article className={`panel table-panel recent-events ${full?'recent-full':''}`} data-testid="recent-events" data-state={state.status}>
    <div className="panel-head"><div className="head-title"><span className="head-icon"><FileText/></span><h2>Recent Events</h2></div>{!full&&<a href="#events">View all <ArrowRight/></a>}</div>
    <table><thead><tr><th>Time</th><th>Event Type</th><th>Device</th><th>Details</th>{full&&<th>Subscriber</th>}</tr></thead>
      <tbody>{rows?.map(e=><tr key={e.id} data-testid="recent-event-row" data-type={e.type}>
        <td><time title={e.timestamp}>{new Date(e.timestamp).toLocaleTimeString('en-GB')}</time></td>
        <td className={`recent-type type-${e.type}`}>{e.type}</td>
        <td><button title={e.device_id} onClick={()=>setSelected(e)}>{short(e.device_id)}</button></td>
        <td title={description(e)}>{description(e)}</td>{full&&<td title={e.subscriber_id}>{short(e.subscriber_id)}</td>}
      </tr>)}</tbody></table>
    {state.status==='loading'&&<p className="recent-note">Loading recent events…</p>}
    {state.status==='success'&&rows?.length===0&&<p className="recent-note">No events observed in this API execution.</p>}
    <p className="recent-note" role="status" title={state.error}>{label}{state.error?` · ${state.error}`:''} · {state.data?.length??'—'} / 100 · Current API execution only</p>
    {selected&&<div className="recent-detail" role="dialog" aria-label="Event details"><button aria-label="Close event details" onClick={()=>setSelected(undefined)}><X/></button><h3>{selected.type}</h3><dl>{Object.entries(selected).map(([key,value])=><div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl></div>}
  </article>;
  return full?<section className="workspace"><header><small>NEXUS CORE LAB / EVENTS</small><h1>Events</h1><p>Last 100 observed events · resets when the API restarts</p></header>{panel}</section>:panel;
}
