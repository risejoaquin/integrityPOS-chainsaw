const { contextBridge } = require('electron')

// Expose any needed APIs to the renderer process
// In a POS system, we might expose printer info, etc.
contextBridge.exposeInMainWorld('electronAPI', {
  platform: process.platform,
  isElectron: true,
})