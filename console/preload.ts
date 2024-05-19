const { contextBridge, app, ipcRenderer } = require("electron");
const { exposeAPI } = require("@synnaxlabs/drift/electron");

exposeAPI();

const GET_VERSION_CMD = "getVersion";

contextBridge.exposeInMainWorld("versionAPI", {
  get: async () => await ipcRenderer.invoke(GET_VERSION_CMD),
});

interface KV {
  get: (key: string) => Promise<string | null>;
  set: (key: string, value: string) => Promise<void>;
  delete: (key: string) => Promise<void>;
}

const KV_API_KEY = "kvAPI";
const KV_GET_CMD = "kvGet";
const KV_SET_CMD = "kvSet";
const KV_DELETE_CMD = "kvDelete";

contextBridge.exposeInMainWorld(KV_API_KEY, {
  get: async (key: string) => await ipcRenderer.invoke(KV_GET_CMD, key),
  set: async (key: string, value: string) =>
    await ipcRenderer.invoke(KV_SET_CMD, key, value),
  delete: async (key: string) => await ipcRenderer.invoke(KV_DELETE_CMD, key),
});
