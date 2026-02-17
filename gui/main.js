const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const fs = require('fs');
const { spawn } = require('child_process');

// ── Logging ──────────────────────────────────────────────────────────
const LOG_PATH = path.join(app.getPath('userData'), 'entropy-tunnel-gui.log');

function log(level, msg) {
  const ts = new Date().toISOString();
  const line = `[${ts}] [${level}] ${msg}\n`;
  try { fs.appendFileSync(LOG_PATH, line); } catch (_) { }
  if (level === 'ERROR') { console.error(line.trim()); }
  else { console.log(line.trim()); }
}

// Catch uncaught errors and write to log
process.on('uncaughtException', (err) => {
  log('FATAL', `Uncaught exception: ${err.stack || err.message}`);
});
process.on('unhandledRejection', (reason) => {
  log('ERROR', `Unhandled rejection: ${reason}`);
});

log('INFO', `EntropyTunnel GUI starting — log: ${LOG_PATH}`);
log('INFO', `Electron ${process.versions.electron}, Node ${process.versions.node}, Chrome ${process.versions.chrome}`);
log('INFO', `Platform: ${process.platform} ${process.arch}`);

// ── App State ────────────────────────────────────────────────────────
let mainWindow;
let clientProcess = null;
const API_BASE = 'http://127.0.0.1:9876';

function createWindow() {
  log('INFO', 'Creating main window');

  const winOpts = {
    width: 420,
    height: 680,
    resizable: false,
    frame: false,
    backgroundColor: '#0a0a14',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  };

  // Only add icon if the file exists
  const iconPath = path.join(__dirname, 'icon.png');
  if (fs.existsSync(iconPath)) {
    winOpts.icon = iconPath;
    log('INFO', 'Icon loaded');
  } else {
    log('WARN', `Icon not found at ${iconPath} — using default`);
  }

  // macOS vibrancy (skip transparent on other platforms — causes crashes)
  if (process.platform === 'darwin') {
    winOpts.vibrancy = 'dark';
    winOpts.transparent = true;
    winOpts.backgroundColor = '#00000000';
  }

  mainWindow = new BrowserWindow(winOpts);
  mainWindow.loadFile(path.join(__dirname, 'index.html'));

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  log('INFO', 'Window created successfully');
}

app.whenReady().then(() => {
  log('INFO', 'App ready');
  createWindow();
}).catch((err) => {
  log('FATAL', `App failed to start: ${err.stack || err.message}`);
});

app.on('window-all-closed', () => {
  log('INFO', 'All windows closed');
  if (clientProcess) {
    clientProcess.kill();
    log('INFO', 'Client process killed');
  }
  app.quit();
});

// ── IPC Handlers ─────────────────────────────────────────────────────
ipcMain.handle('connect', async (event, configPath) => {
  try {
    const clientBin = path.join(__dirname, '..', 'bin', 'entropy-client');
    const cfgPath = configPath || path.join(__dirname, '..', 'configs', 'client-example.yaml');

    log('INFO', `Connecting with binary: ${clientBin}`);
    log('INFO', `Config: ${cfgPath}`);

    if (!fs.existsSync(clientBin)) {
      const errMsg = `Client binary not found: ${clientBin}. Run 'make build' first.`;
      log('ERROR', errMsg);
      return { success: false, error: errMsg };
    }

    const args = ['connect', '-c', cfgPath];
    clientProcess = spawn(clientBin, args, { stdio: 'pipe' });

    clientProcess.stdout.on('data', (data) => {
      const msg = data.toString().trim();
      log('CLIENT', msg);
      if (mainWindow) mainWindow.webContents.send('log', msg);
    });

    clientProcess.stderr.on('data', (data) => {
      const msg = data.toString().trim();
      log('CLIENT-ERR', msg);
      if (mainWindow) mainWindow.webContents.send('log', msg);
    });

    clientProcess.on('error', (err) => {
      log('ERROR', `Client process error: ${err.message}`);
    });

    clientProcess.on('exit', (code) => {
      log('INFO', `Client process exited with code ${code}`);
      if (mainWindow) mainWindow.webContents.send('disconnected', code);
      clientProcess = null;
    });

    await waitForAPI();
    log('INFO', 'API is ready');
    return { success: true };
  } catch (err) {
    log('ERROR', `Connect failed: ${err.stack || err.message}`);
    return { success: false, error: err.message };
  }
});

ipcMain.handle('disconnect', async () => {
  log('INFO', 'Disconnect requested');
  try {
    await fetch(`${API_BASE}/api/disconnect`, { method: 'POST' });
  } catch (_) { }

  if (clientProcess) {
    clientProcess.kill('SIGTERM');
    clientProcess = null;
    log('INFO', 'Client process terminated');
  }
  return { success: true };
});

ipcMain.handle('get-status', async () => {
  try {
    const resp = await fetch(`${API_BASE}/api/status`);
    return await resp.json();
  } catch (_) {
    return { connected: false, status: 'stopped' };
  }
});

ipcMain.handle('toggle-sports-mode', async (event, enabled) => {
  log('INFO', `Sports mode: ${enabled}`);
  try {
    await fetch(`${API_BASE}/api/sports-mode`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    return { success: true };
  } catch (err) {
    log('ERROR', `Sports mode toggle failed: ${err.message}`);
    return { success: false, error: err.message };
  }
});

ipcMain.handle('minimize', () => { if (mainWindow) mainWindow.minimize(); });
ipcMain.handle('close', () => {
  log('INFO', 'Close requested via IPC');
  app.quit();
});

// ── Helpers ──────────────────────────────────────────────────────────
async function waitForAPI(retries = 20) {
  for (let i = 0; i < retries; i++) {
    try {
      const resp = await fetch(`${API_BASE}/api/health`);
      if (resp.ok) return;
    } catch (_) { }
    await new Promise(r => setTimeout(r, 500));
  }
  log('WARN', 'API did not become ready within timeout');
}
