import { useEffect, useMemo, useRef, useState } from 'react';
import * as maplibregl from 'maplibre-gl';
import maplibreWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import type { GeoJSONSource, Map as MapLibreMap, MapOptions } from 'maplibre-gl';
import { MapPin, Minus, Plus, RotateCcw, X } from 'lucide-react';
import { TopologyOverlay, TopologyStatus } from './LiveTopology';
import { MAP_PROVIDER } from './map-provider';
import { buildMapPresentation, mapMode } from './map-presentation';
import { networkCells } from './network-catalog';
import { useTopology } from './topology';
import 'maplibre-gl/dist/maplibre-gl.css';
import './map-topology.css';

maplibregl.setWorkerUrl(maplibreWorkerUrl);

type StyleSpecification = Exclude<MapOptions['style'], string | null | undefined>;
type MapEntityKind = 'cell' | 'device';
type MapSelection = { kind: MapEntityKind; properties: Record<string, unknown> };
type ObservedHandover = { alias: string; sessionId: string; fromCellId: string; toCellId: string };

const observedAssociations = new Map<string, string>();
const emptyLines = (): GeoJSON.FeatureCollection<GeoJSON.LineString> => ({ type: 'FeatureCollection', features: [] });

function handoverCollection(handover?: ObservedHandover): GeoJSON.FeatureCollection<GeoJSON.LineString> {
  if (!handover) return emptyLines();
  const from = networkCells.find(cell => cell.id === handover.fromCellId);
  const to = networkCells.find(cell => cell.id === handover.toCellId);
  if (!from || !to) return emptyLines();
  return {
    type: 'FeatureCollection',
    features: [{
      type: 'Feature', id: handover.sessionId,
      geometry: { type: 'LineString', coordinates: [[from.longitude, from.latitude], [to.longitude, to.latitude]] },
      properties: { ...handover },
    }],
  };
}

const minorRoads = Array.from({ length: 20 }, (_, index) => ({
  d: index % 2 === 0 ? `M ${-80 + index * 53} 0 Q ${180 + index * 19} 210 ${60 + index * 42} 500` : `M 0 ${20 + index * 24} Q 400 ${100 + index * 11} 850 ${10 + index * 23}`,
}));

function Tower({ x, y, type, id }: { x: number; y: number; type: 'lte' | 'g5'; id: string }) {
  return <g className={`tower ${type}`} transform={`translate(${x} ${y})`}>
    <path d="M0 -9L-9 24L0 18L9 24L0 -9M-5 10L5 17M5 10L-5 17" />
    <circle className="emitter" cy="-10" r="3" />
    <path d="M-7 -18Q-15 -10 -7 -2M7 -18Q15 -10 7 -2M-11 -22Q-24 -10 -11 3M11 -22Q24 -10 11 3M-15 -26Q-33 -10 -15 8M15 -26Q33 -10 15 8" />
    <text y="43">{id}</text><text className="tech" y="62">{type === 'lte' ? 'LTE' : '5G'}</text>
  </g>;
}

function StaticFallback({ reason, retry }: { reason: string; retry: () => void }) {
  const [zoom, setZoom] = useState(1);
  return <div className="fallback-map" data-testid="map-fallback" role="img" aria-label="Local fallback map of the simulated NEXUS topology in Campinas">
    <svg viewBox="0 0 850 480" preserveAspectRatio="none" style={{ transform: `scale(${zoom})` }}>
      <defs>
        <radialGradient id="mapBg"><stop stopColor="#10223a" /><stop offset="1" stopColor="#06101d" /></radialGradient>
        <filter id="night-cartography" colorInterpolationFilters="sRGB"><feColorMatrix type="saturate" values="0" /><feComponentTransfer><feFuncR type="linear" slope="-.18" intercept=".20" /><feFuncG type="linear" slope="-.25" intercept=".30" /><feFuncB type="linear" slope="-.30" intercept=".38" /></feComponentTransfer></filter>
        <filter id="glow"><feGaussianBlur stdDeviation="4" result="b" /><feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge></filter>
        <pattern id="blocks" width="72" height="54" patternUnits="userSpaceOnUse" patternTransform="rotate(-11)"><path d="M2 2H67V48H2Z" fill="none" stroke="#19304a" strokeWidth=".7" /><path d="M18 2V48M48 2V48M2 26H67" stroke="#142941" strokeWidth=".55" /></pattern>
      </defs>
      <rect width="850" height="480" fill="url(#mapBg)" /><rect width="850" height="480" fill="url(#blocks)" opacity=".6" />
      <g className="districts"><path d="M0 35L180 0l90 96-54 109L42 180Z" /><path d="M255 0h245l67 105-93 102-208-51Z" /><path d="M570 8l280 38v162l-166 41-119-120Z" /><path d="M0 237l180-69 129 103-59 186L0 480Z" /><path d="M310 207l203-38 113 134-83 165-258-23Z" /><path d="M650 238l200-61v303H582Z" /></g>
      <g className="roads">{minorRoads.map((road, index) => <path d={road.d} key={index} />)}<path className="artery" d="M-30 413 Q180 258 354 281 T880 151" /><path className="artery" d="M60 -20 Q248 172 410 247 T826 503" /><path className="artery2" d="M-20 165 Q219 211 404 173 T884 290" /></g>
      <image href="/maps/campinas.jpg" x="-160" y="-190" width="1250" height="870" preserveAspectRatio="none" filter="url(#night-cartography)" />
      <text className="city" x="422" y="203">CAMPINAS / SP</text>
      {networkCells.map(cell => <Tower key={cell.id} x={cell.x} y={cell.y} type={cell.tech === 'LTE' ? 'lte' : 'g5'} id={cell.id} />)}
      <TopologyOverlay />
      <g className="compass" transform="translate(805 52)"><text y="-23">N</text><circle r="20" /><path d="M0-16L5 2H-5Z M0 16L-5-2H5Z" /></g>
    </svg>
    <div className="map-fallback-message" role="status"><strong>Local map fallback</strong><span>{reason}</span><button type="button" onClick={retry}><RotateCcw /> Retry real map</button></div>
    <span className="location"><MapPin />Campinas, SP - Brazil</span>
    <a className="map-attribution" href="https://commons.wikimedia.org/wiki/File:OSM_Campinas_map.jpg" target="_blank" rel="noreferrer">Fallback: © OpenStreetMap contributors · Sj1mor · CC BY-SA 4.0</a>
    <div className="zoom"><button type="button" aria-label="Zoom fallback map out" onClick={() => setZoom(Math.max(.9, zoom - .05))}><Minus /></button><button type="button" aria-label="Zoom fallback map in" onClick={() => setZoom(Math.min(1.15, zoom + .05))}><Plus /></button></div>
  </div>;
}

function webGLAvailable() {
  try {
    const canvas = document.createElement('canvas');
    return Boolean(window.WebGLRenderingContext && (canvas.getContext('webgl2') || canvas.getContext('webgl')));
  } catch {
    return false;
  }
}

function darkenBaseStyle(map: MapLibreMap) {
  for (const layer of map.getStyle().layers ?? []) {
    try {
      if (layer.type === 'background') map.setPaintProperty(layer.id, 'background-color', '#06111d');
      if (layer.type === 'fill') {
        map.setPaintProperty(layer.id, 'fill-color', layer.id.toLowerCase().includes('water') ? '#071a2a' : '#0c1c28');
        map.setPaintProperty(layer.id, 'fill-opacity', layer.id.toLowerCase().includes('water') ? .78 : .56);
      }
      if (layer.type === 'line') {
        map.setPaintProperty(layer.id, 'line-color', '#28475f');
        map.setPaintProperty(layer.id, 'line-opacity', .58);
      }
      if (layer.type === 'symbol') {
        map.setPaintProperty(layer.id, 'text-color', '#7892a7');
        map.setPaintProperty(layer.id, 'text-halo-color', '#06111d');
        map.setPaintProperty(layer.id, 'text-halo-width', 1);
        map.setPaintProperty(layer.id, 'icon-opacity', .42);
      }
      if (layer.type === 'raster') {
        map.setPaintProperty(layer.id, 'raster-brightness-max', .35);
        map.setPaintProperty(layer.id, 'raster-saturation', -.75);
      }
    } catch {
      // A provider layer may not expose every paint property. Other layers remain usable.
    }
  }
}

function normalizeProviderStyle(style: StyleSpecification): StyleSpecification {
  return {
    ...style,
    layers: style.layers.map(layer => layer.type === 'symbol' && layer.layout?.['text-field']
      ? { ...layer, layout: { ...layer.layout, 'text-font': ['Noto Sans Regular'] } }
      : layer),
  };
}

function popupContent(properties: Record<string, unknown>, kind: 'cell' | 'device') {
  const root = document.createElement('div');
  root.className = 'nexus-map-popup';
  const rows = kind === 'cell'
    ? [['Cell', properties.id], ['Region', properties.region], ['Technology', properties.tech], ['State', properties.status], ['Active sessions', properties.sessionCount], ['Coordinates', 'Illustrative · not infrastructure GPS']]
    : [['Device', properties.alias], ['IMEI', properties.imei], ['Session', properties.sessionId], ['Virtual IP', properties.ipAddress], ['Cell', properties.cellId], ['Position', 'Deterministic presentation coordinate']];
  for (const [label, value] of rows) {
    const line = document.createElement('p');
    const title = document.createElement('strong');
    title.textContent = `${label}: `;
    line.append(title, document.createTextNode(String(value ?? '—')));
    root.append(line);
  }
  return root;
}

function installNexusLayers(
  map: MapLibreMap,
  presentation: ReturnType<typeof buildMapPresentation>,
  handover: GeoJSON.FeatureCollection<GeoJSON.LineString>,
  select: (selection: MapSelection) => void,
) {
  map.addSource('nexus-cells', { type: 'geojson', data: presentation.cells });
  map.addSource('nexus-devices', { type: 'geojson', data: presentation.devices });
  map.addSource('nexus-links', { type: 'geojson', data: presentation.links });
  map.addSource('nexus-handover', { type: 'geojson', data: handover });

  map.addLayer({ id: 'nexus-coverage', type: 'circle', source: 'nexus-cells', paint: {
    'circle-radius': ['interpolate', ['linear'], ['zoom'], 10, 48, 13, 105, 16, 180],
    'circle-color': ['match', ['get', 'tech'], 'LTE', '#329df4', '#b55bf5'],
    'circle-opacity': ['match', ['get', 'activity'], 'ACTIVE', .11, .055],
    'circle-stroke-width': ['interpolate', ['linear'], ['zoom'], 10, .8, 15, 1.5],
    'circle-stroke-opacity': ['match', ['get', 'activity'], 'ACTIVE', .38, .2],
    'circle-stroke-color': ['match', ['get', 'tech'], 'LTE', '#55b6ff', '#cf83ff'],
  }});
  map.addLayer({ id: 'nexus-link-underlay', type: 'line', source: 'nexus-links', paint: {
    'line-color': ['match', ['get', 'state'], 'stale', '#91a5b3', '#2be698'],
    'line-width': 7, 'line-opacity': .13, 'line-blur': 3,
  }});
  map.addLayer({ id: 'nexus-links', type: 'line', source: 'nexus-links', paint: {
    'line-color': ['match', ['get', 'state'], 'stale', '#91a5b3', '#4be5a0'],
    'line-width': 2.2, 'line-opacity': ['match', ['get', 'state'], 'stale', .5, .94], 'line-dasharray': [2, 1.5],
  }});
  map.addLayer({ id: 'nexus-handover-glow', type: 'line', source: 'nexus-handover', paint: {
    'line-color': '#ff8737', 'line-width': 9, 'line-opacity': .18, 'line-blur': 4,
  }});
  map.addLayer({ id: 'nexus-handover', type: 'line', source: 'nexus-handover', paint: {
    'line-color': '#ff963f', 'line-width': 3, 'line-opacity': .95, 'line-dasharray': [3, 1.6],
  }});
  map.addLayer({ id: 'nexus-cell-halos', type: 'circle', source: 'nexus-cells', paint: {
    'circle-radius': ['match', ['get', 'activity'], 'ACTIVE', 30, 26],
    'circle-color': ['match', ['get', 'tech'], 'LTE', '#299dff', '#b44ff3'],
    'circle-opacity': ['match', ['get', 'activity'], 'ACTIVE', .16, .08],
    'circle-blur': .6,
  }});
  map.addLayer({ id: 'nexus-cell-rings', type: 'circle', source: 'nexus-cells', paint: {
    'circle-radius': ['match', ['get', 'tech'], 'LTE', 20, 23],
    'circle-color': '#06131f', 'circle-opacity': .94,
    'circle-stroke-width': ['match', ['get', 'activity'], 'ACTIVE', 3.5, 2.5],
    'circle-stroke-color': ['match', ['get', 'tech'], 'LTE', '#42aaff', '#c060f7'],
  }});
  map.addLayer({ id: 'nexus-cell-cores', type: 'circle', source: 'nexus-cells', paint: {
    'circle-radius': ['match', ['get', 'tech'], 'LTE', 8, 10],
    'circle-color': '#071522', 'circle-stroke-width': 1.5,
    'circle-stroke-color': ['match', ['get', 'tech'], 'LTE', '#9dd5ff', '#e0afff'],
  }});
  map.addLayer({ id: 'nexus-cell-tech', type: 'symbol', source: 'nexus-cells', layout: {
    'text-field': ['match', ['get', 'tech'], 'LTE', 'L', '5'], 'text-size': 11,
    'text-font': ['Noto Sans Regular'], 'text-allow-overlap': true,
  }, paint: { 'text-color': ['match', ['get', 'tech'], 'LTE', '#75c3ff', '#d48aff'], 'text-halo-color': '#06131f', 'text-halo-width': 1 }});
  map.addLayer({ id: 'nexus-cell-labels', type: 'symbol', source: 'nexus-cells', layout: {
    'text-field': ['concat', ['get', 'id'], '\n', ['get', 'region'], ' · ', ['get', 'tech'], ' · ', ['to-string', ['get', 'sessionCount']], ' LINK'],
    'text-size': 12, 'text-line-height': 1.25, 'text-font': ['Noto Sans Regular'],
    'text-offset': ['match', ['get', 'id'], 'CELL-SP-003', ['literal', [2.35, 0]], ['literal', [0, 2.9]]],
    'text-anchor': ['match', ['get', 'id'], 'CELL-SP-003', 'left', 'top'], 'text-allow-overlap': true,
  }, paint: { 'text-color': '#eef7ff', 'text-halo-color': '#04101a', 'text-halo-width': 2 }});
  map.addLayer({ id: 'nexus-device-halos', type: 'circle', source: 'nexus-devices', paint: {
    'circle-radius': 15, 'circle-color': '#30e596', 'circle-opacity': .15, 'circle-blur': .55,
  }});
  map.addLayer({ id: 'nexus-device-rings', type: 'circle', source: 'nexus-devices', paint: {
    'circle-radius': 9, 'circle-color': '#081a20', 'circle-opacity': .96, 'circle-stroke-width': 2,
    'circle-stroke-color': ['match', ['get', 'state'], 'stale', '#9eaab3', '#5bf0a9'],
  }});
  map.addLayer({ id: 'nexus-devices', type: 'circle', source: 'nexus-devices', paint: {
    'circle-radius': 3.8, 'circle-color': ['match', ['get', 'state'], 'stale', '#94a4b0', '#4ce69e'],
    'circle-stroke-width': 1, 'circle-stroke-color': '#e2fff0',
  }});
  map.addLayer({ id: 'nexus-device-labels', type: 'symbol', source: 'nexus-devices', layout: {
    'text-field': ['concat', ['get', 'alias'], ' · ', ['get', 'technology']], 'text-size': 11,
    'text-font': ['Noto Sans Regular'], 'text-offset': [0, 1.45], 'text-anchor': 'top', 'text-allow-overlap': true,
  }, paint: { 'text-color': '#eafff5', 'text-halo-color': '#04101a', 'text-halo-width': 2 }});

  const interactive = ['nexus-devices', 'nexus-cell-rings'];
  for (const layer of interactive) {
    map.on('mouseenter', layer, () => { map.getCanvas().style.cursor = 'pointer'; });
    map.on('mouseleave', layer, () => { map.getCanvas().style.cursor = ''; });
  }
  map.on('click', 'nexus-devices', event => {
    const feature = event.features?.[0];
    if (feature?.properties) {
      select({ kind: 'device', properties: feature.properties });
      new maplibregl.Popup({ closeButton: true, maxWidth: '330px' }).setLngLat(event.lngLat).setDOMContent(popupContent(feature.properties, 'device')).addTo(map);
    }
  });
  map.on('click', 'nexus-cell-rings', event => {
    const feature = event.features?.[0];
    if (feature?.properties) {
      select({ kind: 'cell', properties: feature.properties });
      new maplibregl.Popup({ closeButton: true, maxWidth: '330px' }).setLngLat(event.lngLat).setDOMContent(popupContent(feature.properties, 'cell')).addTo(map);
    }
  });
}

function updateSources(
  map: MapLibreMap,
  presentation: ReturnType<typeof buildMapPresentation>,
  handover: GeoJSON.FeatureCollection<GeoJSON.LineString>,
) {
  (map.getSource('nexus-cells') as GeoJSONSource | undefined)?.setData(presentation.cells);
  (map.getSource('nexus-devices') as GeoJSONSource | undefined)?.setData(presentation.devices);
  (map.getSource('nexus-links') as GeoJSONSource | undefined)?.setData(presentation.links);
  (map.getSource('nexus-handover') as GeoJSONSource | undefined)?.setData(handover);
}

export function MapTopology() {
  const snapshot = useTopology();
  const presentation = useMemo(() => buildMapPresentation(snapshot.entries), [snapshot.entries]);
  const container = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const presentationRef = useRef(presentation);
  presentationRef.current = presentation;
  const [attempt, setAttempt] = useState(0);
  const [mode, setMode] = useState<'loading' | 'ready' | 'fallback'>('loading');
  const [fallbackReason, setFallbackReason] = useState('Interactive map unavailable. Live NEXUS state remains visible on the local map.');
  const [selected, setSelected] = useState<MapSelection>();
  const [handover, setHandover] = useState<ObservedHandover>();
  const handoverData = useMemo(() => handoverCollection(handover), [handover]);
  const handoverRef = useRef(handoverData);
  handoverRef.current = handoverData;

  useEffect(() => {
    let detected: ObservedHandover | undefined;
    const current = new Map<string, string>();
    for (const device of presentation.connected) {
      const previousCellId = observedAssociations.get(device.sessionId);
      current.set(device.sessionId, device.cellId);
      if (previousCellId && previousCellId !== device.cellId) {
        detected = { alias: device.alias, sessionId: device.sessionId, fromCellId: previousCellId, toCellId: device.cellId };
      }
    }
    observedAssociations.clear();
    current.forEach((cellId, sessionId) => observedAssociations.set(sessionId, cellId));
    if (detected) setHandover(detected);
  }, [presentation]);

  useEffect(() => {
    if (!handover) return;
    const timer = window.setTimeout(() => setHandover(undefined), 8000);
    return () => window.clearTimeout(timer);
  }, [handover]);

  useEffect(() => {
    if (!container.current) return;
    const mapContainer = container.current;
    let disposed = false;
    let failed = false;
    let providerErrors = 0;
    let installed = false;
    let map: MapLibreMap | undefined;
    let observer: ResizeObserver | undefined;
    const controller = new AbortController();
    const fail = (reason: string) => {
      if (disposed || failed) return;
      failed = true;
      setFallbackReason(reason);
      setMode('fallback');
      map?.remove();
      mapRef.current = null;
    };
    if (mapMode({ webgl: webGLAvailable() }) === 'fallback') {
      fail('WebGL is unavailable. Live NEXUS state remains visible on the local map.');
      return;
    }
    const initialize = async () => {
      try {
        const styleResponse = await fetch(MAP_PROVIDER.styleURL, { signal: controller.signal });
        if (!styleResponse.ok) throw new Error(`Map style HTTP ${styleResponse.status}`);
        const providerStyle = normalizeProviderStyle(await styleResponse.json() as StyleSpecification);
        if (disposed) return;
        const createdMap = new maplibregl.Map({
        container: mapContainer,
        style: providerStyle,
        center: MAP_PROVIDER.center,
        zoom: MAP_PROVIDER.zoom,
        minZoom: MAP_PROVIDER.minZoom,
        maxZoom: MAP_PROVIDER.maxZoom,
        attributionControl: false,
        cooperativeGestures: true,
        fadeDuration: 150,
      });
        map = createdMap;
        mapRef.current = createdMap;
        createdMap.addControl(new maplibregl.NavigationControl({ showCompass: true, showZoom: true }), 'bottom-right');
        createdMap.addControl(new maplibregl.AttributionControl({ compact: false, customAttribution: 'Telecom positions are simulated' }), 'bottom-left');
        createdMap.on('style.load', () => {
          if (disposed || installed) return;
          installed = true;
          darkenBaseStyle(createdMap);
          installNexusLayers(createdMap, presentationRef.current, handoverRef.current, setSelected);
          setMode('ready');
        });
        createdMap.on('error', () => {
          providerErrors += 1;
          if (!installed || providerErrors >= 6) fail('The map provider could not be loaded. Live NEXUS state remains visible on the local map.');
        });
        observer = new ResizeObserver(() => createdMap.resize());
        observer.observe(mapContainer);
      } catch (error) {
        if (!controller.signal.aborted) fail(error instanceof Error && error.message.startsWith('Map style HTTP')
          ? 'The map provider could not be loaded. Live NEXUS state remains visible on the local map.'
          : 'MapLibre could not initialize. Live NEXUS state remains visible on the local map.');
      }
    };
    void initialize();
    return () => {
      disposed = true;
      controller.abort();
      observer?.disconnect();
      map?.remove();
      mapRef.current = null;
    };
    // A retry is the only intentional reason to create another MapLibre instance.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt]);

  useEffect(() => {
    if (mode === 'ready' && mapRef.current) updateSources(mapRef.current, presentation, handoverData);
  }, [mode, presentation, handoverData]);

  const retry = () => { setMode('loading'); setAttempt(value => value + 1); };
  const activeCells = presentation.cells.features.filter(feature => Number(feature.properties?.sessionCount ?? 0) > 0).length;
  return <div className="map-wrap map-experience" data-map-mode={mode}>
    {mode !== 'fallback' && <div ref={container} className="maplibre-canvas" data-testid="real-map" role="region" aria-label="Interactive map of real Campinas cartography with simulated telecom topology" />}
    {mode === 'fallback' && <StaticFallback reason={fallbackReason} retry={retry} />}
    {mode === 'loading' && <div className="map-loading" role="status">Loading real Campinas cartography…</div>}
    <div className="map-truth" data-testid="map-truth"><strong>REAL CARTOGRAPHY</strong><span>Simulated telecom topology over real Campinas cartography · illustrative coverage</span></div>
    <TopologyStatus />
    <section className="map-ops-hud" data-testid="map-hud" aria-label="Network topology summary">
      <header><span>NETWORK DIGITAL TWIN</span><b>{snapshot.status === 'success' ? 'OBSERVED' : snapshot.status.toUpperCase()}</b></header>
      <div><p><strong>{networkCells.length}</strong><span>Configured cells</span></p><p><strong>{activeCells}</strong><span>Active cells</span></p><p><strong>{presentation.connected.length}</strong><span>Logical links</span></p></div>
      <small>Positions and coverage are simulated</small>
    </section>
    {presentation.connected.length === 0 && snapshot.status === 'success' && <div className="map-empty" data-testid="map-empty"><b>TOPOLOGY READY</b><span>No connected sessions observed</span><small>Configured cells remain visible for operational context</small></div>}
    {handover && <div className="handover-feedback" data-testid="handover-feedback" role="status"><b>HANDOVER OBSERVED</b><span>{handover.alias}</span><strong>{handover.fromCellId} <i>→</i> {handover.toCellId}</strong><small>Logical association changed · session and IP preserved</small></div>}
    {selected && <aside className="map-inspector" data-testid="map-inspector" aria-label="Selected topology entity">
      <header><span>{selected.kind === 'cell' ? 'CELL NODE' : 'CONNECTED DEVICE'}</span><button type="button" aria-label="Close map inspector" onClick={() => setSelected(undefined)}><X /></button></header>
      <strong>{String(selected.properties.id ?? selected.properties.alias ?? 'Topology entity')}</strong>
      {selected.kind === 'cell' ? <dl><dt>Region</dt><dd>{String(selected.properties.region ?? '—')}</dd><dt>Technology</dt><dd>{String(selected.properties.tech ?? '—')}</dd><dt>Operational state</dt><dd>{String(selected.properties.status ?? '—')}</dd><dt>Active sessions</dt><dd>{String(selected.properties.sessionCount ?? 0)}</dd></dl> : <dl><dt>IMEI</dt><dd>{String(selected.properties.imei ?? '—')}</dd><dt>Virtual IP</dt><dd>{String(selected.properties.ipAddress ?? '—')}</dd><dt>Serving cell</dt><dd>{String(selected.properties.cellId ?? '—')}</dd><dt>Technology</dt><dd>{String(selected.properties.technology ?? '—')}</dd><dt>Session</dt><dd>{String(selected.properties.sessionId ?? '—')}</dd></dl>}
      <small>{selected.kind === 'cell' ? 'Configured cell · illustrative map anchor' : 'Real domain state · deterministic simulated position'}</small>
    </aside>}
    <div className="sr-only" role="region" aria-live="polite" aria-label="Connected sessions on topology">
      <p>{presentation.connected.length} connected devices shown.</p>
      {presentation.connected.map(device => <p key={device.id} data-testid="map-device" data-device-id={device.id} data-session-id={device.sessionId} data-cell-id={device.cellId} data-ip={device.ipAddress}>{device.alias}: session {device.sessionId}, IP {device.ipAddress}, cell {device.cellId}, {device.state}.</p>)}
    </div>
  </div>;
}
