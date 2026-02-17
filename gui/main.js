const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const { spawn } = require('child_process');

let mainWindow;
let clientProcess = null;
const API_BASE = 'http://127.0.0.1:9876';

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 420,
    height: 680,
    resizable: false,
    frame: false,
    transparent: true,
    vibrancy: 'dark',
    backgroundColor: '#00000000',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
    icon: path.join(__dirname, 'icon.png'),
  });

  mainWindow.loadFile('index.html');
}

app.whenReady().then(createWindow);

app.on('window-all-closed', () => {
  if (clientProcess) {
    clientProcess.kill();
  }
  app.quit();
});

// IPC handlers
ipcMain.handle('connect', async (event, configPath) => {
  try {
    // Spawn entropy-client as child process
    const clientBin = path.join(__dirname, '..', 'bin', 'entropy-client');
    const args = ['connect', '-c', configPath || path.join(__dirname, '..', 'configs', 'client-example.yaml')];

    clientProcess = spawn(clientBin, args, { stdio: 'pipe' });

    clientProcess.stdout.on('data', (data) => {
      mainWindow.webContents.send('log', data.toString());
    });

    clientProcess.stderr.on('data', (data) => {
      mainWindow.webContents.send('log', data.toString());
    });

    clientProcess.on('exit', (code) => {
      mainWindow.webContents.send('disconnected', code);
      clientProcess = null;
    });

    // Wait for API to be ready
    await waitForAPI();
    return { success: true };
  } catch (err) {
    return { success: false, error: err.message };
  }
});

ipcMain.handle('disconnect', async () => {
  try {
    await fetch(`${API_BASE}/api/disconnect`, { method: 'POST' });
  } catch (_) {}

  if (clientProcess) {
    clientProcess.kill('SIGTERM');
    clientProcess = null;
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
  try {
    await fetch(`${API_BASE}/api/sports-mode`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    return { success: true };
  } catch (err) {
    return { success: false, error: err.message };
  }
});

ipcMain.handle('minimize', () => mainWindow.minimize());
ipcMain.handle('close', () => app.quit());

async function waitForAPI(retries = 20) {
  for (let i = 0; i < retries; i++) {
    try {
      const resp = await fetch(`${API_BASE}/api/health`);
      if (resp.ok) return;
    } catch (_) {}
    await new Promise(r => setTimeout(r, 500));
  }
}
