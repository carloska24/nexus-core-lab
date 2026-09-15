import {storageView} from './storage-api';
import { useState, useEffect, createContext, useContext } from 'react';
import { UrbanMap, ReferenceIcon, SkylineArt } from './ReferenceArt';
import './comparison.css';
import './transport.css';
import { MonitoringProvider, useMonitoring, readLabel } from './monitoring';
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
import { TopologyOverlay, TopologyStatus, ActiveSessionsPreview } from './LiveTopology';
import {
  Activity, Antenna, ArrowRight, BarChart3, Check, CirclePlay, Clock3, Database,
  FileText, Gauge, Home, List, MapPin, Minus, Network, Plus, RadioTower,
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

// MOCK fixtures. No Subscriber/Device/Session integration in Gate 2.
const sessions = [
  ['860010001234567','5511999990001','SP-001','10.45.0.8','00:12:34'],
  ['860010001234568','5511999990002','SP-002','10.45.0.14','00:08:21'],
  ['860010001234569','5511999990003','SP-003','10.45.0.22','00:05:17'],
  ['860010001234570','5511999990004','SP-002','10.45.0.31','00:03:45'],
  ['860010001234571','5511999990005','SP-001','10.45.0.33','00:02:11']
];

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
  return <section className="kpis">{kpis.map(([Icon,label,mockValue,trend,sub,color])=>{
    const subscriberCard=label==='Total Subscribers';
    const deviceCard=label==='Total Devices';
    const real=subscriberCard||deviceCard||label==='Active Sessions'||label==='Total Events';
    const state=subscriberCard?collection:deviceCard?devices:telemetry;
    const value=subscriberCard?collection.data?.length??'—':deviceCard?devices.data?.length??'—':real?telemetry.data?label==='Active Sessions'?telemetry.data.metrics.active_sessions:totalEvents(telemetry.data):'—':mockValue;
    return <article className="kpi" key={label} data-testid={subscriberCard?'subscribers-kpi':deviceCard?'devices-kpi':label==='Active Sessions'?'active-kpi':label==='Total Events'?'events-kpi':undefined} data-state={real?state.status:'configured'} title={real?state.error:'Configured cells · static backend catalogue mirror; not physical health'}>
    <div className={`kpi-icon ${color}`}>{label==='Total Subscribers'?<ReferenceIcon name="users"/>:label==='Total Devices'?<ReferenceIcon name="phone"/>:label==='Total Events'?<ReferenceIcon name="file"/>:<Icon/>}</div><div className="kpi-copy"><span>{label}</span><div><strong>{value}</strong></div><small>{real?readLabel({status:state.status}):sub}</small></div>
  </article>})}</section>;
}

const minorRoads = Array.from({length: 20},(_,i)=>({
  d: i%2===0 ? `M ${-80+i*53} 0 Q ${180+i*19} 210 ${60+i*42} 500` : `M 0 ${20+i*24} Q 400 ${100+i*11} 850 ${10+i*23}`
}));

function Tower({x,y,type,id}:{x:number;y:number;type:'lte'|'g5';id:string}) {
  return <g className={`tower ${type}`} transform={`translate(${x} ${y})`}>
    <path d="M0 -9L-9 24L0 18L9 24L0 -9M-5 10L5 17M5 10L-5 17"/>
    <circle className="emitter" cy="-10" r="3"/>
    <path d="M-7 -18Q-15 -10 -7 -2M7 -18Q15 -10 7 -2M-11 -22Q-24 -10 -11 3M11 -22Q24 -10 11 3M-15 -26Q-33 -10 -15 8M15 -26Q33 -10 15 8"/><text y="43">{id}</text><text className="tech" y="62">{type==='lte'?'LTE':'5G'}</text>
  </g>;
}

function Device({x,y,id}:{x:number;y:number;id:string}) {
  return <g className="device" transform={`translate(${x} ${y})`}><circle r="6"/><text y="20">{id}</text></g>;
}

function Topology() {
  const [zoom,setZoom]=useState(1);
  return <article className="panel topology-panel">
    <div className="panel-head"><div className="head-title"><span className="head-icon"><ReferenceIcon name="topology"/></span><div><h2>Network Topology</h2><p>Observed connections · illustrative positions, not GPS</p></div></div>
      <div className="map-legend"><span><i className="lte"/>LTE Cell</span><span><i className="g5"/>5G Cell</span><span><i className="dev"/>Connected Device</span><span><i className="handover"/>Handover</span></div>
    </div>
    <div className="map-wrap">
      <svg viewBox="0 0 850 480" preserveAspectRatio="none" style={{transform:`scale(${zoom})`}}>
        <defs>
          <radialGradient id="mapBg"><stop stopColor="#10223a"/><stop offset="1" stopColor="#06101d"/></radialGradient>
          <filter id="night-cartography" colorInterpolationFilters="sRGB"><feColorMatrix type="saturate" values="0"/><feComponentTransfer><feFuncR type="linear" slope="-.18" intercept=".20"/><feFuncG type="linear" slope="-.25" intercept=".30"/><feFuncB type="linear" slope="-.30" intercept=".38"/></feComponentTransfer></filter>
          <filter id="glow"><feGaussianBlur stdDeviation="4" result="b"/><feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge></filter>
          <pattern id="blocks" width="72" height="54" patternUnits="userSpaceOnUse" patternTransform="rotate(-11)"><path d="M2 2H67V48H2Z" fill="none" stroke="#19304a" strokeWidth=".7"/><path d="M18 2V48M48 2V48M2 26H67" stroke="#142941" strokeWidth=".55"/></pattern>
        </defs>
        <rect width="850" height="480" fill="url(#mapBg)"/><rect width="850" height="480" fill="url(#blocks)" opacity=".6"/>
        <g className="districts"><path d="M0 35L180 0l90 96-54 109L42 180Z"/><path d="M255 0h245l67 105-93 102-208-51Z"/><path d="M570 8l280 38v162l-166 41-119-120Z"/><path d="M0 237l180-69 129 103-59 186L0 480Z"/><path d="M310 207l203-38 113 134-83 165-258-23Z"/><path d="M650 238l200-61v303H582Z"/></g>
        <g className="roads">{minorRoads.map((r,i)=><path d={r.d} key={i}/>)}<path className="artery" d="M-30 413 Q180 258 354 281 T880 151"/><path className="artery" d="M60 -20 Q248 172 410 247 T826 503"/><path className="artery2" d="M-20 165 Q219 211 404 173 T884 290"/></g>
        <image href="/maps/campinas.jpg" x="-160" y="-190" width="1250" height="870" preserveAspectRatio="none" filter="url(#night-cartography)"/>
        <text className="city" x="422" y="203">CAMPINAS / SP</text>
        {networkCells.map(cell=><Tower key={cell.id} x={cell.x} y={cell.y} type={cell.tech==='LTE'?'lte':'g5'} id={cell.id}/>)}<TopologyOverlay/>
        <g className="compass" transform="translate(805 52)"><text y="-23">N</text><circle r="20"/><path d="M0-16L5 2H-5Z M0 16L-5-2H5Z"/></g>
      </svg>
      <TopologyStatus/><span className="location"><MapPin/>Campinas, SP - Brazil</span><a className="map-attribution" href="https://commons.wikimedia.org/wiki/File:OSM_Campinas_map.jpg" target="_blank" rel="noreferrer">© OpenStreetMap contributors · Sj1mor · CC BY-SA 4.0</a>
      <div className="zoom"><button onClick={()=>setZoom(Math.max(.9,zoom-.05))}><Minus/></button><button onClick={()=>setZoom(Math.min(1.15,zoom+.05))}><Plus/></button></div>
    </div>
  </article>;
}

function Workspace({page}:{page:string}){
  const [query,setQuery]=useState('');
  const [selected,setSelected]=useState<string[]|null>(null);
  const [simEvents,setSimEvents]=useState<string[]>([]);
  const [saved,setSaved]=useState(false);
  const [interval,setIntervalValue]=useState('5');
  const [alerts,setAlerts]=useState(true);
  const rows=sessions;
  const headers=page==='Events'?['Time','Event type','Device','Details']:['Device ID','Subscriber','Cell','IP Address','Duration'];
  return <section className="workspace"><header><div><small>NEXUS CORE LAB / {page.toUpperCase()}</small><h1>{page}</h1><p>{page==='Telemetry'?'Live telemetry · IP pool remains mock':['System','API Status','Database'].includes(page)?'HTTP liveness and storage diagnostics':'Explore the telecom lab · demonstration data'}</p></div></header>
    {['Sessions','Events'].includes(page)?<><input aria-label="Search records" placeholder="Search records…" value={query} onChange={e=>setQuery(e.target.value)}/><div className="workspace-card"><table><thead><tr>{headers.map(h=><th key={h}>{h}</th>)}<th>Details</th></tr></thead><tbody>{rows.filter(r=>r.join(' ').toLowerCase().includes(query.toLowerCase())).map((r,i)=><tr key={i}>{r.map((v,j)=><td key={j}>{v}</td>)}<td><button onClick={()=>setSelected(r)}>Open →</button></td></tr>)}</tbody></table>{!rows.some(r=>r.join(' ').toLowerCase().includes(query.toLowerCase()))&&<p>No matching records.</p>}</div></>:page==='Network'?<div className="network-workspace"><Topology/></div>:page==='Telemetry'?<div className="telemetry-workspace"><ActivityChart/><Donut/><IPPoolCard/></div>:page==='Simulator'?<div className="workspace-card"><h2>Network event simulator</h2><p>Run a local demonstration event. No requests are sent to a backend.</p><div className="sim-actions">{['ATTACH','CELL_HANDOVER','DETACH'].map(t=><button key={t} onClick={()=>setSimEvents(prev=>[`${new Date().toLocaleTimeString()} · ${t} · UE-01 · ${t==='CELL_HANDOVER'?'SP-001 → SP-003':t==='ATTACH'?'Connected to SP-001':'Session terminated'}`,...prev])}>{t}</button>)}<button onClick={()=>setSimEvents([])}>Clear</button></div><ul aria-live="polite">{simEvents.map((e,i)=><li key={i}>{e}</li>)}</ul>{simEvents.length===0&&<p>No simulated events yet.</p>}</div>:page==='Configuration'?<form className="workspace-card config-form" onSubmit={e=>{e.preventDefault();setSaved(true)}}><h2>Display preferences</h2><label>Refresh interval (seconds)<select value={interval} onChange={e=>{setIntervalValue(e.target.value);setSaved(false)}}><option>5</option><option>10</option><option>30</option></select></label><label><input type="checkbox" checked={alerts} onChange={e=>{setAlerts(e.target.checked);setSaved(false)}}/> Show event notifications</label><button type="submit">Save preferences</button>{saved&&<p role="status">Preferences saved for this screen.</p>}</form>:<OperationalStatus page={page}/>}
    {selected&&<div className="modal-shade" onClick={()=>setSelected(null)}><section role="dialog" aria-modal="true" aria-label="Record details" className="workspace-card record-modal" onClick={e=>e.stopPropagation()}><button aria-label="Close details" onClick={()=>setSelected(null)}>×</button><h2>Record details</h2><dl>{selected.map((v,i)=><div key={i}><dt>{headers[i]}</dt><dd>{v}</dd></div>)}</dl></section></div>}
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

