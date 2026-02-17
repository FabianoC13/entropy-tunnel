// EntropyTunnel GUI — Renderer
let connected = false;
let statusPollInterval = null;

const statusRing = document.getElementById('statusRing');
const statusIcon = document.getElementById('statusIcon');
const statusText = document.getElementById('statusText');
const connectBtn = document.getElementById('connectBtn');
const connectBtnText = document.getElementById('connectBtnText');
const statsSection = document.getElementById('statsSection');
const logArea = document.getElementById('logArea');

// Connect / Disconnect
async function toggleConnect() {
    if (connected) {
        setUIState('disconnecting');
        await window.entropy.disconnect();
        setUIState('disconnected');
    } else {
        setUIState('connecting');
        const result = await window.entropy.connect();
        if (result.success) {
            setUIState('connected');
        } else {
            addLog(`Error: ${result.error}`);
            setUIState('disconnected');
        }
    }
}

function setUIState(state) {
    switch (state) {
        case 'disconnected':
            connected = false;
            statusRing.classList.remove('connected');
            connectBtn.classList.remove('connected', 'connecting');
            statusIcon.textContent = '⚡';
            statusText.textContent = 'Disconnected';
            connectBtnText.textContent = 'Connect';
            statsSection.style.display = 'none';
            stopPolling();
            break;

        case 'connecting':
            connectBtn.classList.add('connecting');
            statusIcon.textContent = '🔄';
            statusText.textContent = 'Connecting';
            connectBtnText.textContent = 'Connecting...';
            break;

        case 'connected':
            connected = true;
            statusRing.classList.add('connected');
            connectBtn.classList.remove('connecting');
            connectBtn.classList.add('connected');
            statusIcon.textContent = '🛡️';
            statusText.textContent = 'Protected';
            connectBtnText.textContent = 'Disconnect';
            statsSection.style.display = 'flex';
            startPolling();
            addLog('Tunnel connected. Traffic is encrypted.');
            break;

        case 'disconnecting':
            connectBtn.classList.add('connecting');
            statusText.textContent = 'Disconnecting';
            connectBtnText.textContent = 'Disconnecting...';
            break;
    }
}

// Sports Mode
async function toggleSportsMode() {
    const enabled = document.getElementById('sportsModeToggle').checked;
    await window.entropy.toggleSportsMode(enabled);
    addLog(enabled ? '⚽ Sports Mode ON — Low latency + extra noise' : '⚽ Sports Mode OFF');
}

// Config import
function importConfig() {
    addLog('Config import — place your YAML file in configs/ folder');
}

function scanQR() {
    addLog('QR scanning — coming in v0.2.0');
}

// Status polling
function startPolling() {
    statusPollInterval = setInterval(async () => {
        const status = await window.entropy.getStatus();
        if (status.uptime) {
            document.getElementById('uptime').textContent = status.uptime;
        }
        if (status.bytes_sent !== undefined) {
            document.getElementById('bytesSent').textContent = formatBytes(status.bytes_sent);
            document.getElementById('bytesRecv').textContent = formatBytes(status.bytes_recv);
        }
        if (!status.connected && connected) {
            setUIState('disconnected');
            addLog('Connection lost.');
        }
    }, 2000);
}

function stopPolling() {
    if (statusPollInterval) {
        clearInterval(statusPollInterval);
        statusPollInterval = null;
    }
}

// Logging
function addLog(msg) {
    const entry = document.createElement('div');
    entry.className = 'log-entry';
    const time = new Date().toLocaleTimeString('en-US', { hour12: false });
    entry.textContent = `[${time}] ${msg}`;
    logArea.appendChild(entry);
    logArea.scrollTop = logArea.scrollHeight;

    // Keep last 50 entries
    while (logArea.children.length > 50) {
        logArea.removeChild(logArea.firstChild);
    }
}

// Bytes formatting
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// IPC listeners
window.entropy.onLog((msg) => addLog(msg));
window.entropy.onDisconnected((code) => {
    if (connected) {
        addLog(`Client exited with code ${code}`);
        setUIState('disconnected');
    }
});
