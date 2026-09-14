import { useId } from 'react';
import { Activity, BarChart3 } from 'lucide-react';
import { totalEvents } from './api';
import { readLabel, useMonitoring } from './monitoring';

export function OperationalStatus({ page }: { page: string }) {
  const { health, telemetry } = useMonitoring();
  const database = page === 'Database';
  return <div className="workspace-card"><h2>{database ? 'Database overview' : page === 'API Status' ? 'API status' : 'System overview'}</h2>
    <p>{database ? 'UNKNOWN — no database readiness endpoint is available.' : 'Availability reflects GET /health only, not full system readiness.'}</p>
    <dl><dt>Status</dt><dd data-testid="workspace-status">{database ? 'UNKNOWN' : health.status === 'success' ? 'HTTP ONLINE' : health.status === 'loading' ? 'Checking…' : 'HTTP OFFLINE'}</dd>
      {!database && <><dt>Service</dt><dd>{health.data?.service ?? '—'}</dd><dt>Last successful health check</dt><dd>{health.receivedAt ? new Date(health.receivedAt).toLocaleString() : '—'}</dd><dt>Telemetry</dt><dd>{readLabel(telemetry)}</dd><dt>Requests since process start</dt><dd>{telemetry.data?.requests_total ?? '—'}</dd></>}
    </dl>
  </div>;
}

export function ActivityChart() {
  const { telemetry, observations } = useMonitoring();
  const gradient = useId();
  const max = Math.max(4, ...observations.map(point => point.active_sessions));
  const ceiling = Math.ceil(max / 4) * 4;
  const first = observations[0];
  const last = observations[observations.length - 1];
  const start = first ? Date.parse(first.timestamp) : 0;
  const span = last ? Math.max(5000, Date.parse(last.timestamp) - start) : 5000;
  const positions = observations.map(point => ({
    x: 38 + (Date.parse(point.timestamp) - start) / span * 450,
    y: 140 - point.active_sessions / ceiling * 120,
    gap: point.gap,
  }));
  const path = positions.map((point, index) => `${index === 0 || point.gap ? 'M' : 'L'}${point.x} ${point.y}`).join(' ');
  const hasGap = positions.some(point => point.gap);
  const time = (timestamp: string) => new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  return <article className="panel analytic" data-testid="activity" data-state={telemetry.status}>
    <div className="panel-head"><div className="head-title"><span className="head-icon"><Activity/></span><div><h2>Session Activity</h2><p>{readLabel(telemetry)} · observation since opening</p></div></div><div className="chart-legend"><span><i className="green"/>Active Sessions</span></div></div>
    <svg className="line-chart corrected-chart" viewBox="0 0 500 160" preserveAspectRatio="none" aria-label={`Observed active sessions: ${observations.length} samples`}>
      <defs><linearGradient id={gradient} x1="0" y1="0" x2="0" y2="1"><stop stopColor="#23c983" stopOpacity=".3"/><stop offset="1" stopColor="#23c983" stopOpacity=".03"/></linearGradient></defs>
      <g className="grid">{[0,1,2,3,4].map(step=><g key={step}><path d={`M38 ${140-step*30}H488`}/><text x="14" y={144-step*30}>{ceiling*step/4}</text></g>)}{[38,113,188,263,338,413,488].map(x=><path key={x} d={`M${x} 20V140`}/>)}</g>
      {positions.length >= 2 ? <>{!hasGap && <path d={`${path} L${positions[positions.length-1].x} 140L38 140Z`} fill={`url(#${gradient})`}/>}<path d={path} className="active-line"/>{positions.map((point,index)=><circle key={index} cx={point.x} cy={point.y} r="2.6" fill="#3ddb85"/>)}<text x="72" y="157">{time(first.timestamp)}</text><text x="450" y="157">{time(last.timestamp)}</text></> : <text x="263" y="85">{telemetry.status === 'error' ? 'Telemetry unavailable' : 'Collecting telemetry…'}</text>}
    </svg>
  </article>;
}

export function Donut() {
  const { telemetry } = useMonitoring();
  const data = telemetry.data;
  const total = data ? totalEvents(data) : undefined;
  const types = [
    ['ATTACH', data?.events_total.attach, '#36cb72'],
    ['CELL_HANDOVER', data?.events_total.cell_handover, '#f14c52'],
    ['DETACH', data?.events_total.detach, '#ffba54'],
    ['STALE_DISCONNECT', data?.events_total.stale_disconnect, '#7396ba'],
  ] as const;
  let cumulative = 0;
  const slices = types.map(([,value,color]) => {
    const start = cumulative;
    cumulative += total ? (value ?? 0) / total * 100 : 0;
    return `${color} ${start}% ${cumulative}%`;
  });
  return <article className="panel analytic donut-panel" data-testid="events-donut" data-state={telemetry.status}>
    <div className="panel-head"><div className="head-title"><span className="head-icon"><BarChart3/></span><div><h2>Events by Type</h2><p title={telemetry.error}>{readLabel(telemetry)} · process counters</p></div></div></div>
    <div className="donut-content"><div className="donut" style={{ background: total ? `conic-gradient(${slices.join(',')})` : '#243748' }}><span><strong>{total ?? '—'}</strong>{total === 0 ? 'No events' : 'Events'}</span></div><div className="donut-list">{types.map(([label,value,color])=><p key={label} title={`${label}: ${value ?? 'unavailable'}`}><i style={{background:color}}/>{label}<b>{total === undefined ? '—' : `${total === 0 ? 0 : ((value ?? 0)/total*100).toFixed(1)}%`}</b></p>)}</div></div>
  </article>;
}
