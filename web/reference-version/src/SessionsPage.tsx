import { useEffect, useRef, useState } from 'react';
import { RefreshCw, RadioTower } from 'lucide-react';
import { useDevices } from './devices';
import { useSubscribers } from './subscribers';
import { readLabel, type ReadState } from './monitoring';
import type { Device } from './device-api';
import { attachSession, detachSession, getActiveSession, handoverSession, sessionCells, sessionError, type Session } from './session-api';
import './subscribers.css';
import './sessions.css';

function DeviceSession({ device }: { device: Device }) {
  const [state, setState] = useState<ReadState<Session | null>>({ status: 'loading' });
  const [busy, setBusy] = useState(false);
  const [cell, setCell] = useState<string>(sessionCells[0].id);
  const [notice, setNotice] = useState('');
  const [retry, setRetry] = useState(0);
  const mounted = useRef(false);
  const session = state.data;
  const connected = session?.status === 'CONNECTED';
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    setState(previous => ({ ...previous, status: 'loading', error: undefined }));
    getActiveSession(device.id, controller.signal).then(data => {
      if (!controller.signal.aborted) setState({ status: 'success', data, receivedAt: Date.now() });
    }).catch(error => {
      if (!controller.signal.aborted) setState(previous => ({ ...previous, status: previous.data === undefined ? 'error' : 'stale', error: sessionError(error) }));
    });
    return () => controller.abort();
  }, [device.id, retry]);
  const act = async (action: 'attach' | 'handover' | 'detach') => {
    if (busy || state.status !== 'success') return;
    setBusy(true); setNotice('');
    try {
      const data = await (action === 'attach' ? attachSession(device.id, cell) : action === 'handover' ? handoverSession(session!.id, cell) : detachSession(session!.id));
      if (mounted.current) {
        setState({ status: 'success', data, receivedAt: Date.now() });
        setNotice(`Server confirmed ${action}: ${data.status}.`);
      }
    } catch (error) {
      // Writes can have reached the server even if the reply was lost. Require a fresh read.
      if (mounted.current) setState(previous => ({ ...previous, status: previous.data === undefined ? 'error' : 'stale', error: sessionError(error) }));
    } finally { if (mounted.current) setBusy(false); }
  };
  return <div className="workspace-card session-detail" data-testid="session-panel" data-state={state.status}>
    <div className="session-heading"><div><h2><RadioTower size={21}/>Device session</h2><p>IMEI {device.imei}</p></div><button disabled={busy || state.status === 'loading'} onClick={() => { setNotice(''); setRetry(n => n + 1); }}><RefreshCw size={15}/>Refresh session</button></div>
    {state.status === 'loading' && <p role="status">Checking the active session…</p>}
    {state.error && <p className="subscriber-error" role="alert">{state.error} Refresh before another operation.</p>}
    {state.status === 'stale' && <p>Last confirmed snapshot; current server state is unknown.</p>}
    {state.status === 'success' && session === null && <p data-testid="session-empty">No active session for this device.</p>}
    {session && <>
      <span className={`subscriber-badge ${connected ? 'status-ACTIVE' : 'status-DEACTIVATED'}`} data-testid="session-status">{session.status}</span>
      <dl className="subscriber-details session-fields" data-testid="session-details">
        {Object.entries({ 'Session UUID': session.id, 'Device UUID': session.device_id, 'Subscriber UUID': session.subscriber_id, 'Cell': session.cell_id, 'IP address': session.ip_address, 'Attached at': session.attached_at, 'Updated at': session.updated_at, ...(session.closed_at ? { 'Closed at': session.closed_at } : {}), ...(session.disconnect_reason ? { 'Disconnect reason': session.disconnect_reason } : {}) }).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}
      </dl>
    </>}
    <fieldset className="session-controls" disabled={busy || state.status !== 'success'}>
      <legend>{connected ? 'Manage connection' : 'Connect to a cell'}</legend>
      <label>Network cell<select aria-label="Session cell" value={cell} onChange={e => setCell(e.target.value)}>{sessionCells.map(c => <option key={c.id} value={c.id}>{c.id} · {c.name} · {c.tech}</option>)}</select></label>
      <div className="subscriber-actions">
        {connected ? <><button disabled={cell === session.cell_id} onClick={() => void act('handover')}>Handover</button><button className="destructive" onClick={() => void act('detach')}>Detach</button></> : <button disabled={device.status !== 'REGISTERED'} onClick={() => void act('attach')}>Attach</button>}
      </div>
    </fieldset>
    {device.status !== 'REGISTERED' && <p>Only REGISTERED devices can attach.</p>}
    {busy && <p role="status">Waiting for server confirmation…</p>}
    {notice && <p className="subscriber-success" role="status">{notice}</p>}
  </div>;
}
export function SessionsPage() {
  const devices = useDevices();
  const subscribers = useSubscribers();
  const [deviceID, setDeviceID] = useState('');
  const device = devices.collection.data?.find(d => d.id === deviceID);
  return <section className="workspace subscribers-workspace sessions-workspace">
    <header><div><small>NEXUS CORE LAB / SESSIONS</small><h1>Sessions</h1><p>Connection management · select a device</p></div></header>
    <div className="subscriber-toolbar"><label className="session-selector">Device<select aria-label="Session device" value={deviceID} onChange={e => setDeviceID(e.target.value)}><option value="">Select a registered device</option>{devices.collection.data?.map(d => <option key={d.id} value={d.id}>{d.imei} · {d.technology} · {subscribers.collection.data?.find(s => s.id === d.subscriber_id)?.msisdn ?? d.subscriber_id}</option>)}</select></label><button disabled={devices.refreshing} onClick={() => void devices.refresh()}><RefreshCw size={15}/>Refresh devices</button></div>
    <p className="session-source">Devices: {readLabel(devices.collection)} · Sessions are queried individually on selection or refresh.</p>
    {devices.collection.error && <p className="subscriber-error" role="alert">{devices.collection.error}</p>}
    {device ? <DeviceSession key={device.id} device={device}/> : <div className="workspace-card"><h2>Select a device</h2><p>{devices.collection.status === 'success' && devices.collection.data?.length === 0 ? 'No devices available. Register a device first.' : 'Choose a device to view its active connection or attach it to the network.'}</p><a href="#devices">Open Devices →</a></div>}
  </section>;
}
