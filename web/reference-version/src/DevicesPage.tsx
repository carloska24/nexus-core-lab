import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Plus, RefreshCw, Smartphone, X } from 'lucide-react';
import { useDevices } from './devices';
import { useSubscribers } from './subscribers';
import { readLabel, type ReadState } from './monitoring';
import { deviceError, getDevice, getDevicesBySubscriber, registerDevice, validIMEI, type Device, type DeviceTechnology } from './device-api';
import './subscribers.css';
import './devices.css';

const date = (value: string) => new Date(value).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' });
const Badge = ({ status }: { status: Device['status'] }) => <span className={`subscriber-badge device-status-${status}`} data-testid="device-status">{status}</span>;

function DeviceDialog({ id, onClose }: { id: string | null; onClose: () => void }) {
  const { accept } = useDevices();
  const subscribers = useSubscribers();
  const dialog = useRef<HTMLDialogElement>(null);
  const mounted = useRef(false);
  const [device, setDevice] = useState<Device>();
  const [loading, setLoading] = useState(id !== null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [busy, setBusy] = useState(false);
  const [imei, setIMEI] = useState('');
  const [technology, setTechnology] = useState<DeviceTechnology>('LTE');
  const [subscriberID, setSubscriberID] = useState('');
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    mounted.current = true;
    dialog.current?.showModal();
    return () => { mounted.current = false; };
  }, []);
  useEffect(() => {
    if (!id) return;
    const controller = new AbortController();
    setLoading(true); setError('');
    getDevice(id, controller.signal).then(value => {
      if (!controller.signal.aborted) { setDevice(value); setLoading(false); }
    }).catch(cause => {
      if (!controller.signal.aborted) { setError(deviceError(cause)); setLoading(false); }
    });
    return () => controller.abort();
  }, [id, retry]);
  const eligible = subscribers.collection.data?.filter(sub => sub.status === 'ACTIVE') ?? [];
  const canSelect = subscribers.collection.status === 'success' && !subscribers.refreshing;
  const register = async (event: FormEvent) => {
    event.preventDefault();
    if (busy) return;
    setError(''); setSuccess('');
    if (!validIMEI(imei)) { setError('IMEI must contain exactly 15 digits.'); return; }
    if (!canSelect || !eligible.some(sub => sub.id === subscriberID)) { setError('Select an ACTIVE subscriber from the current collection.'); return; }
    setBusy(true);
    try {
      const result = await registerDevice({ subscriber_id: subscriberID, imei, technology });
      accept(result);
      if (mounted.current) { setDevice(result); setSuccess('Device registered. Server status: ' + result.status); }
    } catch (cause) {
      if (mounted.current) setError(deviceError(cause));
      // A subscriber may have changed status in another client. Backend remains authoritative.
      void subscribers.refresh();
    } finally { if (mounted.current) setBusy(false); }
  };
  const owner = subscribers.collection.data?.find(sub => sub.id === device?.subscriber_id);
  return <dialog ref={dialog} className="subscriber-dialog workspace-card device-dialog" aria-labelledby="device-dialog-title" onCancel={event => { if (busy) event.preventDefault(); else onClose(); }}>
    <div className="subscriber-dialog-head"><div><small>DEVICE REGISTRY</small><h2 id="device-dialog-title">{device || id ? 'Device details' : 'Register device'}</h2></div><button type="button" aria-label="Close device dialog" disabled={busy} onClick={onClose}><X size={20}/></button></div>
    {loading && <p role="status">Loading device…</p>}
    {device ? <>
      <div className="subscriber-identity"><strong>{device.imei}</strong><Badge status={device.status}/></div>
      <dl className="subscriber-details" data-testid="device-details">
        <div><dt>UUID</dt><dd>{device.id}</dd></div><div><dt>IMEI</dt><dd>{device.imei}</dd></div>
        <div><dt>Technology</dt><dd>{device.technology}</dd></div><div><dt>Status</dt><dd>{device.status}</dd></div>
        <div className="device-owner-id"><dt>Subscriber ID</dt><dd>{device.subscriber_id}</dd></div>
        {owner && <><div><dt>Subscriber MSISDN</dt><dd>{owner.msisdn}</dd></div><div><dt>Subscriber IMSI</dt><dd>{owner.imsi}</dd></div></>}
        <div><dt>Created At</dt><dd>{date(device.created_at)}<small>{device.created_at}</small></dd></div>
        <div><dt>Updated At</dt><dd>{date(device.updated_at)}<small>{device.updated_at}</small></dd></div>
      </dl>
    </> : !id && <form className="subscriber-provision" onSubmit={register} noValidate>
      <p>Register a device for an active subscriber.</p>
      <label>Subscriber<select aria-label="Device subscriber" value={subscriberID} disabled={busy || !canSelect} onChange={event => setSubscriberID(event.target.value)}>
        <option value="">Select an ACTIVE subscriber</option>
        {subscribers.collection.data?.map(sub => <option key={sub.id} value={sub.id} disabled={sub.status !== 'ACTIVE'}>{sub.msisdn} · {sub.imsi} · {sub.status}</option>)}
      </select></label>
      {!canSelect && <p role="status">Subscriber collection: {readLabel(subscribers.collection)}. Refresh before selecting.</p>}
      {canSelect && eligible.length === 0 && <p className="device-empty-owners">No active subscribers available. <a href="#subscribers" onClick={onClose}>Create or activate a Subscriber</a> to register a device.</p>}
      <button type="button" disabled={busy || subscribers.refreshing} onClick={() => void subscribers.refresh()}>Refresh subscribers</button>
      <label>IMEI<input aria-label="Register IMEI" inputMode="numeric" autoComplete="off" placeholder="15 digits" value={imei} disabled={busy} onChange={event => setIMEI(event.target.value)}/></label>
      <label>Technology<select aria-label="Device technology" value={technology} disabled={busy} onChange={event => setTechnology(event.target.value as DeviceTechnology)}><option>LTE</option><option>5G</option></select></label>
      <button type="submit" disabled={busy || !canSelect || eligible.length === 0}>Register</button>
    </form>}
    {error && <div className="subscriber-error" role="alert">{error}{id && <button type="button" onClick={() => setRetry(value => value + 1)}>Retry details</button>}</div>}
    {busy && <p role="status">Waiting for server confirmation…</p>}
    {success && <p className="subscriber-success" role="status">{success}</p>}
  </dialog>;
}

export function DevicesPage() {
  const { collection, refreshing, refresh } = useDevices();
  const subscribers = useSubscribers();
  const [query, setQuery] = useState('');
  const [modal, setModal] = useState<{ id: string | null } | null>(null);
  const [subscriberID, setSubscriberID] = useState('');
  const [filtered, setFiltered] = useState<ReadState<Device[]>>({ status: 'loading' });
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    if (!subscriberID) return;
    const controller = new AbortController();
    getDevicesBySubscriber(subscriberID, controller.signal).then(data => {
      if (!controller.signal.aborted) setFiltered({ status: 'success', data, receivedAt: Date.now() });
    }).catch(error => {
      if (!controller.signal.aborted) setFiltered(previous => ({ ...previous, status: previous.data === undefined ? 'error' : 'stale', error: deviceError(error) }));
    });
    return () => controller.abort();
  }, [subscriberID, collection.receivedAt, retry]);
  const source = subscriberID ? filtered : collection;
  const owners = new Map(subscribers.collection.data?.map(sub => [sub.id, sub]));
  const rows = source.data?.filter(device => {
    const owner = owners.get(device.subscriber_id);
    return [device.imei, device.technology, device.status, device.subscriber_id, owner?.msisdn ?? '', owner?.imsi ?? ''].some(value => value.toLowerCase().includes(query.toLowerCase()));
  });
  return <section className="workspace subscribers-workspace devices-workspace" data-testid="devices-page" data-state={collection.status}>
    <header><div><small>NEXUS CORE LAB / DEVICES</small><h1>Devices</h1><p>Device registry · real API data</p></div><button className="provision-action" onClick={() => setModal({ id: null })}><Plus size={17}/>Register device</button></header>
    <div className="subscriber-toolbar"><div className="device-filters"><input aria-label="Search devices" placeholder="IMEI, technology, status or subscriber…" value={query} onChange={event => setQuery(event.target.value)}/>
      <select aria-label="Filter devices by subscriber" value={subscriberID} onChange={event => { setSubscriberID(event.target.value); setFiltered({ status: 'loading' }); }}><option value="">All subscribers</option>{subscribers.collection.data?.map(sub => <option value={sub.id} key={sub.id}>{sub.msisdn} · {sub.imsi}</option>)}</select>
    </div><button disabled={refreshing} onClick={() => { void refresh(); setRetry(value => value + 1); }}><RefreshCw size={15}/>{refreshing ? 'Refreshing…' : 'Refresh'}</button></div>
    <div className="subscriber-collection-meta"><span><Smartphone size={15}/><strong data-testid="device-total">{collection.data?.length ?? '—'}</strong> total devices{source.data && <span> · {rows?.length} shown</span>}</span><span>{refreshing ? 'Refreshing collection…' : readLabel(collection)}</span></div>
    {collection.error && <p className="subscriber-error" role="alert">{collection.error} {collection.data ? 'Showing the last known global collection.' : 'Global collection unavailable.'}</p>}
    {subscriberID && <p role="status">Subscriber filter: {readLabel(filtered)}{filtered.error && ` · ${filtered.error}`}</p>}
    <div className="workspace-card subscriber-table device-table"><table><thead><tr>{['IMEI', 'Technology', 'Status', 'Subscriber', 'Created At', 'Updated At', 'Details'].map(label => <th key={label}>{label}</th>)}</tr></thead><tbody>
      {rows?.map(device => { const owner = owners.get(device.subscriber_id); return <tr key={device.id} data-testid="device-row"><td>{device.imei}</td><td><span className={`device-tech tech-${device.technology}`}>{device.technology}</span></td><td><Badge status={device.status}/></td><td className="device-owner" title={device.subscriber_id}>{owner ? <>{owner.msisdn}<small>{owner.imsi}</small></> : device.subscriber_id}</td><td title={device.created_at}>{date(device.created_at)}</td><td title={device.updated_at}>{date(device.updated_at)}</td><td><button aria-label={`Open device ${device.imei}`} onClick={() => setModal({ id: device.id })}>Open →</button></td></tr>; })}
    </tbody></table>
    {source.status === 'loading' && <p role="status">Loading devices…</p>}
    {source.status === 'success' && rows?.length === 0 && <p>{!subscriberID && !query && collection.data?.length === 0 ? 'No devices registered yet.' : 'No matching devices.'}</p>}
    {source.status === 'error' && <p>Devices unavailable. Use Refresh to retry.</p>}
    {source.status === 'stale' && <p>Last known snapshot. Current server state is unavailable.</p>}
    </div>
    {modal && <DeviceDialog key={modal.id ?? 'register'} id={modal.id} onClose={() => setModal(null)}/>}
  </section>;
}
