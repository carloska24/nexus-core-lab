import {storageView} from './storage-api';
import { useState, useEffect, createContext, useContext } from 'react';
import { ReferenceIcon, SkylineArt } from './ReferenceArt';
import './comparison.css';
import './transport.css';
import { MonitoringProvider, useMonitoring, readLabel, snapshotLabel } from './monitoring';
import { totalEvents } from './api';
import { ActivityChart, Donut, OperationalStatus } from './LivePanels';
import { SubscribersProvider, useSubscribers } from './subscribers';
import { SubscribersPage } from './SubscribersPage';
import { DevicesProvider, useDevices } from './devices';
import { DevicesPage } from './DevicesPage';
import { SessionsPage } from './SessionsPage';
import { networkCells } from './network-catalog';
import { RecentEventsProvider } from './recent-events';
import { RecentEvents } from './RecentEvents';
import { IPPoolCard } from './IPPoolCard';
import { TopologyProvider } from './topology';
import { ActiveSessionsPreview } from './LiveTopology';
import { MapTopology } from './MapTopology';
import {
  Activity, Antenna, ArrowRight, BarChart3, Check, CirclePlay, Clock3, Database,
  FileText, Gauge, Home, List, Network, RadioTower,
  Server, Settings, Smartphone, Users, Wifi, X
} from 'lucide-react';

const Navigation = createContext({page:'Overview',go:(_page:string)=>{}});
const nav = [
  [Home, 'Overview'], [Users, 'Subscribers'], [Smartphone, 'Devices'],
  [Network, 'Sessions'], [RadioTower, 'Network'], [List, 'Events'],
  [BarChart3, 'Telemetry'], [CirclePlay, 'Simulator']
] as const;

// Network Cells are configured (static backend catalogue mirror), not discovered health.
const kpis = [
  [Users, 'Total Subscribers', '—', '', 'Provisioned in system', 'blue'],
  [Smartphone, 'Total Devices', '—', '', 'Registered devices', 'blue'],
  [Wifi, 'Active Sessions', '—', '', 'Currently connected', 'blue'],
  [RadioTower, 'Network Cells', String(networkCells.length), '', 'Configured cells · Campinas', 'blue'],
  [Server, 'Total Events', '—', '', 'Process counters', 'purple']
] as const;

function Brand() {
  return <><div className="brand"><div>NE<span>X</span>US</div><small>CORE LAB</small></div><svg className="brand-outline" viewBox="0 0 249 94"><path d="M248 0V77L225 92H0"/></svg></>;
}

function Sidebar() {
  const {health,storage}=useMonitoring();
  const storageDisplay=storageView(storage);
  const {page,go}=useContext(Navigation);
  return <aside className="sidebar">
    <Brand />
    <nav>{nav.map(([Icon,label])=><a href={`#${label.toLowerCase()}`} onClick={()=>go(label)} className={page===label?'active':''} key={label}><Icon/><span>{label}</span></a>)}</nav>
    <div className="side-divider" />
    <div className="system-links">
      {[[Settings,'System'],[Gauge,'API Status'],[Database,'Database'],[Settings,'Configuration']].map(([Icon,label]:any)=><a key={label} href={`#${label.toLowerCase().replaceAll(' ','-')}`} className={page===label?'active':''} onClick={()=>go(label)}><Icon/><span>{label}</span>{['API Status','Database'].includes(label)&&<i data-status={label==='Database'?storageDisplay.dot:health.status==='success'?'online':health.status==='loading'?'loading':'offline'} title={label==='Database'?`Storage: ${storageDisplay.label}`:`API: ${health.status}`}/>}</a>)}
    </div>
    <div className="skyline" aria-hidden="true">
      <SkylineArt/>
    </div>
    <footer><strong>NEXUS CORE LAB</strong><span>CAMPINAS / SP</span><em>“Connecting Ideas<br/>to a Smarter Tomorrow”</em></footer>
  </aside>;
}

function Topbar() {
  const {go}=useContext(Navigation);
  const {health,storage}=useMonitoring();
  const storageDisplay=storageView(storage);
  const status=health.status==='success'?'online':health.status==='loading'?'loading':'offline';
  return <header className="topbar">
    <div><p>Telecom Network Simulator &amp; Core Lab</p><small>Build&nbsp;&nbsp;•&nbsp;&nbsp;Learn&nbsp;&nbsp;•&nbsp;&nbsp;Simulate&nbsp;&nbsp;•&nbsp;&nbsp;Explore</small></div>
    <div className="top-actions"><span className="online" data-status={status} data-testid="system-status" title="HTTP availability only; not full system readiness"><i data-status={status}/> SYSTEM {status.toUpperCase()}</span><time>{health.receivedAt?new Date(health.receivedAt).toLocaleString('pt-BR'):'Checking API…'}</time><button className="settings-action" aria-label="Open configuration" onClick={()=>go('Configuration')}><Settings className="gear"/></button><button className="infra" onClick={()=>go('API Status')} title={`API: HTTP health. Storage: ${storageDisplay.label}`}>API <i data-status={status}/> <span data-testid="storage-status" data-state={storage.status}>Storage {storageDisplay.label}</span> <i data-status={storageDisplay.dot}/></button></div>
  </header>;
}

function Kpis() {
  const {telemetry}=useMonitoring();
  const {collection}=useSubscribers();
  const {collection:devices}=useDevices();
  return <section className="kpis">{kpis.map(([Icon,label,configuredValue,trend,sub,color])=>{
    const subscriberCard=label==='Total Subscribers';
    const deviceCard=label==='Total Devices';
    const real=subscriberCard||deviceCard||label==='Active Sessions'||label==='Total Events';
    const state=subscriberCard?collection:deviceCard?devices:telemetry;
    const value=subscriberCard?collection.data?.length??'—':deviceCard?devices.data?.length??'—':real?telemetry.data?label==='Active Sessions'?telemetry.data.metrics.active_sessions:totalEvents(telemetry.data):'—':configuredValue;
    return <article className="kpi" key={label} data-testid={subscriberCard?'subscribers-kpi':deviceCard?'devices-kpi':label==='Active Sessions'?'active-kpi':label==='Total Events'?'events-kpi':undefined} data-state={real?state.status:'configured'} title={real?(state.error || ((subscriberCard||deviceCard) ? `Collection snapshot${state.receivedAt ? ` · Last refreshed ${new Date(state.receivedAt).toLocaleString()}` : ''}. No continuous polling.` : label==='Total Events' ? 'Counters from the current API process; not persisted history.' : undefined)):'Configured cells · static backend catalogue mirror; not physical health'}>
    <div className={`kpi-icon ${color}`}>{label==='Total Subscribers'?<ReferenceIcon name="users"/>:label==='Total Devices'?<ReferenceIcon name="phone"/>:label==='Total Events'?<ReferenceIcon name="file"/>:<Icon/>}</div><div className="kpi-copy"><span>{label}</span><div><strong>{value}</strong></div><small>{subscriberCard||deviceCard?snapshotLabel({status:state.status}):label==='Total Events'&&state.status==='success'?'Current API process':real?readLabel({status:state.status}):sub}</small></div>
  </article>})}</section>;
}

function Topology() {
  return <article className="panel topology-panel">
    <div className="panel-head"><div className="head-title"><span className="head-icon"><ReferenceIcon name="topology"/></span><div><h2>Network Topology</h2><p>Real Campinas map · simulated telecom positions · no GPS</p></div></div>
      <div className="map-legend"><span><i className="lte"/>LTE Cell</span><span><i className="g5"/>5G Cell</span><span><i className="dev"/>Connected Device</span></div>
    </div>
    <MapTopology />
  </article>;
}

function Workspace({page}:{page:string}){
  const [simEvents,setSimEvents]=useState<string[]>([]);
  return <section className={`workspace${page==='Network'?' network-page':''}`}><header><div><small>NEXUS CORE LAB / {page.toUpperCase()}</small><h1>{page==='Simulator'?'Local Event Sandbox':page}</h1><p>{page==='Telemetry'?'Current API counters · browser observations · authoritative IP pool snapshot':['System','API Status','Database'].includes(page)?'HTTP liveness and storage diagnostics':page==='Network'?'Real session associations · configured cells':page==='Simulator'?'Frontend-only simulation':page==='Configuration'?'Effective settings · read-only':'Explore the telecom lab'}</p></div></header>
    {page==='Network'?<div className="network-workspace"><Topology/></div>:page==='Telemetry'?<div className="telemetry-workspace"><ActivityChart/><Donut/><IPPoolCard/></div>:page==='Simulator'?<div className="workspace-card"><h2>Simulate locally</h2><p>These sandbox actions only change this page. They do not send backend requests or change Subscribers, Devices or Sessions.</p><p>For the real Hero Flow, use Subscribers → Devices → Sessions. The separate <code>cmd/simulator</code> CLI also runs the real flow over HTTP; this page does not start it.</p><div className="sim-actions">{['ATTACH','CELL_HANDOVER','DETACH'].map(t=><button key={t} onClick={()=>setSimEvents(prev=>[`${new Date().toLocaleTimeString()} · ${t} · UE-01 · ${t==='CELL_HANDOVER'?'SP-001 → SP-003':t==='ATTACH'?'Connected to SP-001':'Session terminated'}`,...prev])}>{t}</button>)}<button onClick={()=>setSimEvents([])}>Clear</button></div><ul aria-live="polite">{simEvents.map((e,i)=><li key={i}>{e}</li>)}</ul>{simEvents.length===0&&<p>No simulated events yet.</p>}</div>:page==='Configuration'?<div className="workspace-card config-form"><h2>Effective polling intervals</h2><p>Read-only · not configurable in this demo. Delays apply after each request or polling round completes.</p><dl data-testid="effective-polling"><dt>Telemetry</dt><dd>~5 s</dd><dt>Topology</dt><dd>~5 s · Overview and Network only</dd><dt>IP Pool</dt><dd>~5 s · while its panel is open</dd><dt>Recent Events</dt><dd>~5 s</dd><dt>Storage</dt><dd>~10 s</dd><dt>Health</dt><dd>~10 s</dd></dl><p>Subscribers and Devices: collection snapshots, refreshed on load, manual refresh or related operations. No continuous polling.</p><p>Event notifications: not implemented in this demo.</p></div>:<OperationalStatus page={page}/>}
  </section>;
}

export default function App(){
  const pages=['Overview',...nav.slice(1).map(n=>n[1]),'System','API Status','Database','Configuration'];
  const readPage=()=>pages.find(p=>p.toLowerCase().replace(/ /g,'-')===window.location.hash.slice(1))||'Overview';
  const [page,setPage]=useState(readPage);
  const [scale,setScale]=useState(()=>Math.min(window.innerWidth/1536,window.innerHeight/1024,1));
  useEffect(()=>{const resize=()=>setScale(Math.min(window.innerWidth/1536,window.innerHeight/1024,1));const route=()=>setPage(readPage());window.addEventListener('resize',resize);window.addEventListener('hashchange',route);return()=>{window.removeEventListener('resize',resize);window.removeEventListener('hashchange',route)}},[]);
  const go=(next:string)=>{setPage(next);window.location.hash=next.toLowerCase().replace(/ /g,'-')};
  return <MonitoringProvider><RecentEventsProvider><SubscribersProvider><DevicesProvider><TopologyProvider enabled={page==='Overview'||page==='Network'}><Navigation.Provider value={{page,go}}><div className="fit-shell" style={{width:1536*scale,height:1024*scale}}><div className="app" style={{transform:`scale(${scale})`,transformOrigin:'top left'}}><Sidebar/><main><Topbar/>{page==='Overview'?<div className="dashboard"><Kpis/><section className="center"><Topology/><div className="right-tables"><ActiveSessionsPreview/><RecentEvents/></div></section><section className="bottom"><ActivityChart/><Donut/><IPPoolCard/></section></div>:page==='Subscribers'?<SubscribersPage/>:page==='Devices'?<DevicesPage/>:page==='Sessions'?<SessionsPage/>:page==='Events'?<RecentEvents full/>:<Workspace key={page} page={page}/>}</main></div></div></Navigation.Provider></TopologyProvider></DevicesProvider></SubscribersProvider></RecentEventsProvider></MonitoringProvider>
}

