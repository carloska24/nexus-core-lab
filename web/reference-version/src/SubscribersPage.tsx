import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Plus, RefreshCw, Users, X } from 'lucide-react';
import { readLabel } from './monitoring';
import { useSubscribers } from './subscribers';
import { activateSubscriber, deactivateSubscriber, getSubscriber, getSubscriberByIMSI, provisionSubscriber,
  subscriberError, suspendSubscriber, validIMSI, validMSISDN, type Subscriber, type SubscriberStatus } from './subscriber-api';
import './subscribers.css';

const date = (value:string) => new Date(value).toLocaleString('en-GB', {dateStyle:'medium',timeStyle:'short'});
function StatusBadge({status}:{status:SubscriberStatus}) {
  return <span className={`subscriber-badge status-${status}`} data-testid="subscriber-status">{status}</span>;
}

function SubscriberDialog({id,onClose}:{id:string|null;onClose:()=>void}) {
  const {accept,refresh}=useSubscribers();
  const dialog=useRef<HTMLDialogElement>(null);
  const mounted=useRef(true);
  const readController=useRef<AbortController>();
  const [subscriber,setSubscriber]=useState<Subscriber>();
  const [loading,setLoading]=useState(id!==null);
  const [readError,setReadError]=useState('');
  const [writeError,setWriteError]=useState('');
  const [success,setSuccess]=useState('');
  const [busy,setBusy]=useState(false);
  const [imsi,setImsi]=useState('');
  const [msisdn,setMsisdn]=useState('');
  const [reason,setReason]=useState('');
  const [reload,setReload]=useState(0);
  const detailID=id??subscriber?.id;
  useEffect(()=>{
    mounted.current=true;
    dialog.current?.showModal();
    return ()=>{mounted.current=false;readController.current?.abort();};
  },[]);
  useEffect(()=>{
    if(!detailID)return;
    const controller=new AbortController();readController.current=controller;
    setLoading(true);setReadError('');
    getSubscriber(detailID,controller.signal).then(data=>{
      if(!controller.signal.aborted){setSubscriber(data);setLoading(false);}
    }).catch(error=>{
      if(!controller.signal.aborted){setReadError(subscriberError(error));setLoading(false);}
    });
    return ()=>controller.abort();
  },[detailID,reload]);

  const provision=async(event:FormEvent)=>{
    event.preventDefault();if(busy)return;
    setWriteError('');setSuccess('');
    if(!validIMSI(imsi)){setWriteError('IMSI must contain exactly 15 digits.');return;}
    if(!validMSISDN(msisdn)){setWriteError('MSISDN must contain 10–15 digits, begin with 1–9, and may start with +.');return;}
    setBusy(true);
    try{
      const result=await provisionSubscriber({imsi,msisdn});
      accept(result);
      if(mounted.current){setSubscriber(result);setSuccess('Subscriber provisioned. Server status: '+result.status);}
    }catch(error){if(mounted.current)setWriteError(subscriberError(error));}
    finally{if(mounted.current)setBusy(false);}
  };
  const transition=async(action:'activate'|'suspend'|'deactivate')=>{
    if(!subscriber||busy)return;
    setBusy(true);setWriteError('');setSuccess('');
    try{
      const result=await (action==='activate'?activateSubscriber(subscriber.id):action==='suspend'?suspendSubscriber(subscriber.id,reason||undefined):deactivateSubscriber(subscriber.id,reason||undefined));
      accept(result);
      if(mounted.current){setSubscriber(result);setReason('');setSuccess('Server confirmed: '+result.status);}
    }catch(error){
      if(mounted.current){setWriteError(subscriberError(error));setReload(value=>value+1);}
      void refresh();
    }finally{if(mounted.current)setBusy(false);}
  };
  return <dialog ref={dialog} className="subscriber-dialog workspace-card" aria-labelledby="subscriber-dialog-title" onCancel={event=>{if(busy)event.preventDefault();else onClose();}}>
    <div className="subscriber-dialog-head"><div><small>SUBSCRIBER REGISTRY</small><h2 id="subscriber-dialog-title">{subscriber?'Subscriber details':id?'Subscriber details':'Provision subscriber'}</h2></div><button type="button" aria-label="Close subscriber dialog" disabled={busy} onClick={onClose}><X size={20}/></button></div>
    {loading&&<p role="status">Loading subscriber…</p>}
    {readError&&<div className="subscriber-error" role="alert">{readError}<button type="button" onClick={()=>setReload(value=>value+1)}>Retry details</button></div>}
    {subscriber?<>
      <div className="subscriber-identity"><strong>{subscriber.msisdn}</strong><StatusBadge status={subscriber.status}/></div>
      <dl className="subscriber-details" data-testid="subscriber-details">
        <div><dt>UUID</dt><dd>{subscriber.id}</dd></div><div><dt>IMSI</dt><dd>{subscriber.imsi}</dd></div><div><dt>MSISDN</dt><dd>{subscriber.msisdn}</dd></div>
        <div><dt>Created At</dt><dd title={subscriber.created_at}>{date(subscriber.created_at)}<small>{subscriber.created_at}</small></dd></div>
        <div><dt>Updated At</dt><dd title={subscriber.updated_at}>{date(subscriber.updated_at)}<small>{subscriber.updated_at}</small></dd></div>
        {subscriber.suspension_reason&&<div><dt>Suspension reason</dt><dd>{subscriber.suspension_reason}</dd></div>}
        {subscriber.deactivation_reason&&<div><dt>Deactivation reason</dt><dd>{subscriber.deactivation_reason}</dd></div>}
      </dl>
      {subscriber.status==='DEACTIVATED'?<p className="terminal-notice">DEACTIVATED is terminal. No further lifecycle operations are available.</p>:<div className="subscriber-lifecycle">
        <label>Reason <span>(optional, for suspend/deactivate)</span><input aria-label="Lifecycle reason" value={reason} disabled={busy||loading||!!readError} onChange={event=>setReason(event.target.value)}/></label>
        <div className="subscriber-actions">
          {['PENDING_ACTIVATION','SUSPENDED'].includes(subscriber.status)&&<button type="button" disabled={busy||loading||!!readError} onClick={()=>void transition('activate')}>Activate</button>}
          {['ACTIVE','SUSPENDED'].includes(subscriber.status)&&<button type="button" disabled={busy||loading||!!readError} onClick={()=>void transition('suspend')}>Suspend</button>}
          <button type="button" className="destructive" disabled={busy||loading||!!readError} onClick={()=>void transition('deactivate')}>Deactivate permanently</button>
        </div>
      </div>}
    </>:id===null&&<form onSubmit={provision} noValidate className="subscriber-provision">
      <p>Provision a subscriber in PENDING_ACTIVATION. Identities are sent as strings.</p>
      <label>IMSI<input aria-label="Provision IMSI" autoComplete="off" inputMode="numeric" value={imsi} onChange={event=>setImsi(event.target.value)} disabled={busy} placeholder="15 digits"/></label>
      <label>MSISDN<input aria-label="Provision MSISDN" autoComplete="off" inputMode="tel" value={msisdn} onChange={event=>setMsisdn(event.target.value)} disabled={busy} placeholder="+5519999999999"/></label>
      <button type="submit" disabled={busy}>Provision</button>
    </form>}
    {busy&&<p role="status">Waiting for server confirmation…</p>}
    {writeError&&<p className="subscriber-error" role="alert">{writeError}</p>}
    {success&&<p className="subscriber-success" role="status">{success}</p>}
  </dialog>;
}

export function SubscribersPage(){
  const {collection,refreshing,refresh}=useSubscribers();
  const [query,setQuery]=useState('');
  const [modal,setModal]=useState<{id:string|null}|null>(null);
  const [searching,setSearching]=useState(false);
  const [searchError,setSearchError]=useState('');
  const searchController=useRef<AbortController>();
  useEffect(()=>()=>searchController.current?.abort(),[]);
  const exactSearch=async()=>{
    if(!validIMSI(query)||searching)return;
    const controller=new AbortController();searchController.current=controller;
    setSearching(true);setSearchError('');
    try{
      const result=await getSubscriberByIMSI(query,controller.signal);
      if(!controller.signal.aborted){setModal({id:result.id});void refresh();}
    }catch(error){if(!controller.signal.aborted)setSearchError(subscriberError(error));}
    finally{if(!controller.signal.aborted)setSearching(false);}
  };
  const rows=collection.data?.filter(subscriber=>[subscriber.imsi,subscriber.msisdn,subscriber.status].some(value=>value.toLowerCase().includes(query.toLowerCase())))
    .sort((a,b)=>a.created_at.localeCompare(b.created_at)||a.id.localeCompare(b.id));
  return <section className="workspace subscribers-workspace" data-testid="subscribers-page" data-state={collection.status}>
    <header><div><small>NEXUS CORE LAB / SUBSCRIBERS</small><h1>Subscribers</h1><p>Subscriber registry · real API data</p></div><button type="button" className="provision-action" onClick={()=>setModal({id:null})}><Plus size={17}/>Provision subscriber</button></header>
    <div className="subscriber-toolbar"><div className="subscriber-search"><input aria-label="Search subscribers" value={query} onChange={event=>{setQuery(event.target.value);setSearchError('');}} placeholder="Search IMSI, MSISDN or status…"/><button type="button" disabled={!validIMSI(query)||searching} onClick={()=>void exactSearch()}>{searching?'Searching…':'Find exact IMSI'}</button></div><button type="button" disabled={refreshing} onClick={()=>void refresh()}><RefreshCw size={15}/>{refreshing?'Refreshing…':'Refresh'}</button></div>
    {searchError&&<p className="subscriber-error" role="alert">{searchError}</p>}
    <div className="subscriber-collection-meta"><span><Users size={15}/> <strong data-testid="subscriber-total">{collection.data?.length??'—'}</strong> total subscribers</span><span>{refreshing?'Refreshing collection…':readLabel(collection)}{collection.receivedAt&&` · ${new Date(collection.receivedAt).toLocaleTimeString()}`}</span></div>
    {collection.error&&<p className="subscriber-error" role="alert">{collection.error} {collection.data!==undefined?'Showing the last known collection.':'No collection is available. Use Refresh to retry.'}</p>}
    <div className="workspace-card subscriber-table"><table><thead><tr>{['MSISDN','IMSI','Status','Created At','Updated At','Details'].map(label=><th key={label}>{label}</th>)}</tr></thead><tbody>
      {rows?.map(subscriber=><tr key={subscriber.id} data-testid="subscriber-row"><td>{subscriber.msisdn}</td><td>{subscriber.imsi}</td><td><StatusBadge status={subscriber.status}/></td><td title={subscriber.created_at}>{date(subscriber.created_at)}</td><td title={subscriber.updated_at}>{date(subscriber.updated_at)}</td><td><button type="button" onClick={()=>setModal({id:subscriber.id})} aria-label={`Open subscriber ${subscriber.imsi}`}>Open →</button></td></tr>)}
    </tbody></table>
    {collection.status==='loading'&&<p role="status">Loading subscribers…</p>}
    {collection.status==='success'&&rows?.length===0&&<p>{collection.data?.length===0?'No subscribers provisioned yet.':'No matching subscribers.'}</p>}
    {collection.status==='stale'&&rows?.length===0&&<p>No matching records in the last known snapshot. Current server state is unavailable.</p>}
    {collection.status==='error'&&<p>Subscribers unavailable.</p>}
    </div>
    {modal&&<SubscriberDialog key={modal.id??'provision'} id={modal.id} onClose={()=>setModal(null)}/>}
  </section>;
}
