import { useState } from 'react';
import { ArrowRight, Server } from 'lucide-react';
import { networkCells } from './network-catalog';
import { useSubscribers } from './subscribers';
import { sessionDuration, useTopology, type TopologyEntry } from './topology';
import './topology.css';

function description(entry: TopologyEntry, alias: string, subscriber?: { msisdn: string; status: string }) {
  const cell = networkCells.find(c => c.id === entry.session?.cell_id);
  return [alias, `Device UUID: ${entry.device.id}`, `IMEI: ${entry.device.imei}`, `Device technology: ${entry.device.technology}`, `Device status: ${entry.device.status}`, `Connection: ${entry.status.toUpperCase()}`, `Subscriber UUID: ${entry.device.subscriber_id}`, subscriber ? `MSISDN: ${subscriber.msisdn} · ${subscriber.status}` : '', entry.session ? `Session UUID: ${entry.session.id}\nCell: ${entry.session.cell_id} · ${cell?.tech ?? 'unconfigured'}\nVirtual IP: ${entry.session.ip_address}\nDuration at observation: ${sessionDuration(entry.session.attached_at, entry.receivedAt)}\nAttached: ${entry.session.attached_at}` : '', entry.error ?? '', 'Illustrative placement, not GPS'].filter(Boolean).join('\n');
}
export function TopologyOverlay() {
  const snapshot = useTopology();
  const { collection } = useSubscribers();
  const [selected, setSelected] = useState<string>();
  const aliases = new Map(snapshot.entries.map((e, i) => [e.device.id, `D-${String(i + 1).padStart(2, '0')}`]));
  const points = networkCells.flatMap((cell, cellIndex) => {
    const group = snapshot.entries.filter(e => e.session?.cell_id === cell.id && (e.status === 'connected' || e.status === 'stale'));
    const columns = Math.min(5, Math.ceil(Math.sqrt(group.length * 2)));
    const rows = Math.ceil(group.length / columns);
    const left = [85, 290, 480][cellIndex], top = [337, 115, 344][cellIndex];
    return group.map((entry, i) => ({ entry, cell,
      x: group.length === 1 ? [171, 337, 557][cellIndex] : left + (i % columns + .5) * 280 / columns,
      y: top + Math.floor(i / columns) * Math.min(32, 85 / Math.max(1, rows - 1)),
      labels: group.length <= 15,
    }));
  });
  return <g className="live-topology-overlay">
    <g className="links">{points.map(({ entry, cell, x, y }) => <path key={entry.device.id} data-testid="topology-link" data-device-id={entry.device.id} data-cell-id={cell.id} data-state={entry.status} className={entry.status === 'stale' ? 'stale-link' : ''} d={`M${x} ${y} Q${x} ${cell.y + 40} ${cell.x} ${cell.y + 18}`}/>)}</g>
    {points.map(({ entry, x, y, labels }) => {
      const alias = aliases.get(entry.device.id)!;
      const owner = collection.data?.find(s => s.id === entry.device.subscriber_id);
      const label = description(entry, alias, owner);
      return <g key={entry.device.id} role="button" tabIndex={0} aria-label={label} className={`device live-device ${entry.status}`} transform={`translate(${x} ${y})`} data-testid="topology-device" data-device-id={entry.device.id} data-session-id={entry.session?.id} data-ip={entry.session?.ip_address} data-state={entry.status} onClick={() => setSelected(selected === entry.device.id ? undefined : entry.device.id)} onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setSelected(selected === entry.device.id ? undefined : entry.device.id); } if (e.key === 'Escape') setSelected(undefined); }}>
        <title>{label}</title><circle r="6"/>{labels && <text y="20">{alias}{entry.status === 'stale' ? ' ?' : ''}</text>}
        {selected === entry.device.id && <foreignObject x={Math.min(10, 555 - x)} y={Math.min(25, 245 - y)} width="280" height="228"><div className="topology-tooltip">{label.split('\n').map((line, i) => <div key={i}>{line}</div>)}</div></foreignObject>}
      </g>;
    })}
  </g>;
}
export function TopologyStatus() {
  const snapshot = useTopology();
  const { collection } = useSubscribers();
  const [expanded, setExpanded] = useState(false);
  const connected = snapshot.entries.filter(e => e.status === 'connected').length;
  const disconnected = snapshot.entries.filter(e => e.status === 'disconnected').length;
  const uncertain = snapshot.entries.length - connected - disconnected;
  return <div className="topology-state" data-testid="topology-state" data-state={snapshot.status}>
    <button aria-label="Topology device states" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>{snapshot.status.toUpperCase()} · {connected} connected · {disconnected} without session{uncertain > 0 && ` · ${uncertain} unknown/stale`}</button>
    {expanded && <div className="topology-register" role="region" aria-label="Topology device registry"><b>Known Devices · illustrative positions, not GPS</b>{snapshot.error && <p>{snapshot.error}</p>}{snapshot.entries.length === 0 && <p>{snapshot.status === 'success' ? 'No registered devices.' : 'Device collection unavailable or loading.'}</p>}{snapshot.entries.map((entry, i) => <div key={entry.device.id} data-testid="topology-registry-row" data-state={entry.status} title={description(entry, `D-${i + 1}`, collection.data?.find(s => s.id === entry.device.subscriber_id))}><strong>{entry.device.imei}</strong> · {entry.device.technology} · {entry.device.status}<br/><span>{entry.status.toUpperCase()}{entry.session && ` · ${entry.session.cell_id} · ${entry.session.ip_address}`}</span>{entry.error && <small>{entry.error}</small>}</div>)}</div>}
  </div>;
}
export function ActiveSessionsPreview() {
  const snapshot = useTopology();
  const { collection } = useSubscribers();
  const rows = snapshot.entries.filter(e => e.status === 'connected' && e.session);
  return <article className="panel table-panel live-sessions-preview" data-testid="topology-preview" data-state={snapshot.status}>
    <div className="panel-head"><div className="head-title"><span className="head-icon"><Server/></span><h2>Active Sessions ({rows.length})</h2></div><a href="#sessions">Manage <ArrowRight/></a></div>
    <table><thead><tr>{['','IMEI','Subscriber','Cell','IP Address','Duration'].map(h => <th key={h}>{h}</th>)}</tr></thead><tbody>{rows.slice(0, 5).map(entry => <tr key={entry.device.id} data-testid="live-session-row"><td><i className="status-dot"/></td><td title={entry.device.id}>{entry.device.imei}</td><td title={entry.session!.subscriber_id}>{collection.data?.find(s => s.id === entry.session!.subscriber_id)?.msisdn ?? '—'}</td><td title={entry.session!.cell_id}>{entry.session!.cell_id.replace('CELL-', '')}</td><td>{entry.session!.ip_address}</td><td>{sessionDuration(entry.session!.attached_at, snapshot.receivedAt)}</td></tr>)}</tbody></table>
    <p className="preview-state">{snapshot.status === 'success' ? rows.length ? `${Math.min(5, rows.length)} of ${rows.length} observed · topology snapshot` : 'No active sessions observed.' : `${snapshot.status.toUpperCase()} · only confirmed connections shown`}</p>
  </article>;
}
