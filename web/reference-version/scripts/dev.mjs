import { spawn, spawnSync } from 'node:child_process';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer as createNetServer } from 'node:net';
import { parseArgs } from 'node:util';

const root = fileURLToPath(new URL('../', import.meta.url));
const repository = resolve(root, '../..');
let child, vite, temporary;
let stopping = false;
async function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  try {
    await vite?.close();
    if (child?.pid && child.exitCode === null && child.signalCode === null) {
      const exited = new Promise(done => child.once('exit', done));
      if (process.platform === 'win32') {
        spawnSync('taskkill', ['/PID', String(child.pid), '/T', '/F'], { windowsHide: true, stdio: 'ignore' });
      } else {
        child.kill('SIGTERM');
        const timer = setTimeout(() => child.kill('SIGKILL'), 5000);
        timer.unref();
      }
      await exited;
    }
    if (temporary) await rm(temporary, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
  } finally { process.exit(code); }
}
process.on('SIGINT', () => void stop());
process.on('SIGTERM', () => void stop());

function start(command, args, env = process.env) {
  child = spawn(command, args, { cwd: repository, env, stdio: 'inherit', windowsHide: true });
  return child;
}
async function assertPortFree(port) {
  await new Promise((done, fail) => {
    const server = createNetServer();
    server.once('error', () => fail(new Error(`Porta da API ${port} ocupada. Encerre a API anterior ou defina PORT com outra porta.`)));
    server.listen(port, '127.0.0.1', () => server.close(done));
  });
}

try {
  const { values } = parseArgs({ options: { port: { type: 'string' }, host: { type: 'string' }, strictPort: { type: 'boolean' } } });
  const port = Number(process.env.PORT || 8080);
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error('PORT deve estar entre 1 e 65535.');
  await assertPortFree(port);
  temporary = await mkdtemp(join(tmpdir(), 'nexus-dev-'));
  const executable = join(temporary, process.platform === 'win32' ? 'api.exe' : 'api');
  console.log('[dev] Compilando a API Go…');
  const build = start('go', ['build', '-o', executable, './cmd/api']);
  await new Promise((done, fail) => {
    build.once('error', fail);
    build.once('exit', code => code === 0 ? done() : fail(new Error(`Build Go falhou (${code}).`)));
  });
  if (stopping) process.exit(0);
  const target = `http://127.0.0.1:${port}`;
  process.env.NEXUS_API_TARGET = target;
  console.log(`[dev] API: ${target}. Armazenamento: ${process.env.DATABASE_URL ? 'PostgreSQL configurado' : 'memória (dados temporários)'}.`);
  const api = start(executable, [], { ...process.env, PORT: String(port) });
  api.once('error', error => { console.error(error.message); void stop(1); });
  api.once('exit', code => { if (!stopping) { console.error(`[dev] API encerrada (${code}).`); void stop(1); } });
  const deadline = Date.now() + 30000;
  while (!stopping) {
    try {
      const response = await fetch(`${target}/health`, { signal: AbortSignal.timeout(1000) });
      const health = await response.json();
      if (response.ok && health.status === 'ok' && health.service === 'nexus-core-lab') break;
    } catch { /* Wait for this API to start listening. */ }
    if (Date.now() > deadline) throw new Error('API não ficou pronta em 30 segundos.');
    await new Promise(done => setTimeout(done, 250));
  }
  if (!stopping) {
    const { createServer } = await import('vite');
    vite = await createServer({ root, server: { port: values.port ? Number(values.port) : undefined, host: values.host, strictPort: values.strictPort } });
    await vite.listen();
    vite.printUrls();
    console.log('[dev] Frontend e backend prontos. Ctrl+C encerra ambos.');
  }
} catch (error) {
  console.error(`[dev] ${error.message}`);
  await stop(1);
}
