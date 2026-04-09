const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  apiCall: (method, url, data) => ipcRenderer.invoke('api-call', { method, url, data }),
  healthCheck: () => ipcRenderer.invoke('health-check')
});