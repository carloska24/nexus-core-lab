import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { buildMapPresentation, mapMode, presentationCoordinate } from '../src/map-presentation.ts';
import { networkCells } from '../src/network-catalog.ts';

const device = { id: 'device-1', subscriber_id: 'subscriber-1', imei: '860010001234567', technology: '5G', status: 'REGISTERED', created_at: '2026-09-17T00:00:00Z', updated_at: '2026-09-17T00:00:00Z' };
const session = { id: 'session-1', device_id: device.id, subscriber_id: device.subscriber_id, cell_id: 'CELL-SP-001', ip_address: '10.45.0.4', status: 'CONNECTED', attached_at: '2026-09-17T00:00:00Z', updated_at: '2026-09-17T00:00:00Z' };
const entry = { device, session, status: 'connected', receivedAt: Date.now() };

test('configured Cells have explicit presentation coordinates', () => {
  assert.equal(networkCells.length, 3);
  for (const cell of networkCells) {
    assert.ok(Number.isFinite(cell.longitude) && Number.isFinite(cell.latitude));
    assert.match(cell.id, /^CELL-SP-00[1-3]$/);
  }
  assert.equal(buildMapPresentation([]).cells.features.length, 3);
});

test('CONNECTED Session produces one inspectable Device and one Cell link', () => {
  const result = buildMapPresentation([entry]);
  assert.equal(result.devices.features.length, 1);
  assert.equal(result.links.features.length, 1);
  assert.equal(result.connected[0].sessionId, session.id);
  assert.equal(result.connected[0].ipAddress, session.ip_address);
});

test('attach, handover and detach follow the authoritative Session snapshot', () => {
  assert.equal(buildMapPresentation([{ ...entry, session: null, status: 'disconnected' }]).connected.length, 0);
  const attached = buildMapPresentation([entry]).connected[0];
  const handed = buildMapPresentation([{ ...entry, session: { ...session, cell_id: 'CELL-SP-002' } }]).connected[0];
  assert.equal(handed.cellId, 'CELL-SP-002');
  assert.equal(handed.sessionId, attached.sessionId);
  assert.equal(handed.ipAddress, attached.ipAddress);
  assert.notDeepEqual(handed.coordinate, attached.coordinate);
  assert.equal(buildMapPresentation([{ ...entry, session: { ...session, status: 'DISCONNECTED' }, status: 'disconnected' }]).connected.length, 0);
});

test('presentation placement is deterministic for the same Session and Cell', () => {
  assert.deepEqual(presentationCoordinate('session-1:device-1', 'CELL-SP-003'), presentationCoordinate('session-1:device-1', 'CELL-SP-003'));
});

test('fallback selection covers WebGL, initialization and provider failures', () => {
  assert.equal(mapMode({ webgl: true }), 'interactive');
  assert.equal(mapMode({ webgl: false }), 'fallback');
  assert.equal(mapMode({ webgl: true, initializationFailed: true }), 'fallback');
  assert.equal(mapMode({ webgl: true, providerFailed: true }), 'fallback');
});

test('map implementation never requests browser geolocation', () => {
  const root = path.resolve(import.meta.dirname, '../src');
  const source = ['MapTopology.tsx', 'map-presentation.ts', 'map-provider.ts'].map(file => fs.readFileSync(path.join(root, file), 'utf8')).join('\n');
  assert.doesNotMatch(source, /navigator\s*\.\s*geolocation|GeolocateControl/);
});
