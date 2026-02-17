const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('entropy', {
    connect: (configPath) => ipcRenderer.invoke('connect', configPath),
    disconnect: () => ipcRenderer.invoke('disconnect'),
    getStatus: () => ipcRenderer.invoke('get-status'),
    toggleSportsMode: (enabled) => ipcRenderer.invoke('toggle-sports-mode', enabled),
    minimize: () => ipcRenderer.invoke('minimize'),
    close: () => ipcRenderer.invoke('close'),
    onLog: (callback) => ipcRenderer.on('log', (e, msg) => callback(msg)),
    onDisconnected: (callback) => ipcRenderer.on('disconnected', (e, code) => callback(code)),
});
