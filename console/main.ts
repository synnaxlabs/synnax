const { app, shell, BrowserWindow, ipcMain, dialog } = require("electron");
const { dirname, join } = require("path");
const { MAIN_WINDOW } = require("@synnaxlabs/drift");
const { listenOnMain } = require("@synnaxlabs/drift/electron");
const { fileURLToPath } = require("url");
const fs = require("fs");
const { Mutex } = require("async-mutex");
const { autoUpdater } = require("electron-updater");

const __dirname = dirname(fileURLToPath(import.meta.url));

process.env.APP_ROOT = join(__dirname, "..");
export const MAIN_DIST = join(process.env.APP_ROOT, "dist-electron");
export const RENDERER_DIST = join(process.env.APP_ROOT, "dist");
export const VITE_DEV_SERVER_URL = process.env.VITE_DEV_SERVER_URL;

const loadRender = (win) => {
  if (process.env.VITE_DEV_SERVER_URL) win.loadURL(process.env.VITE_DEV_SERVER_URL);
  else win.loadFile(join(RENDERER_DIST, "index.html"));
};

if (process.defaultApp) {
  if (process.argv.length >= 2) {
    app.setAsDefaultProtocolClient("synnax", process.execPath, [process.argv[1]]);
  } else {
    app.setAsDefaultProtocolClient("synnax");
  }
}

function createWindow() {
  const preload = join(MAIN_DIST, "preload.js");
  const mainWindow = new BrowserWindow({
    width: 900,
    height: 670,
    title: MAIN_WINDOW,
    show: false,
    frame: false,
    minWidth: 625,
    minHeight: 375,
    autoHideMenuBar: true,
    webPreferences: {
      preload,
      sandbox: false,
    },
  });
  loadRender(mainWindow);
  listenOnMain({
    mainWindow,
    createWindow: (props) => {
      const win = new BrowserWindow({
        ...props,
        frame: false,
        webPreferences: { preload, sandbox: false },
      });
      loadRender(win);
      return win;
    },
  });

  mainWindow.on("ready-to-show", () => mainWindow.show());

  mainWindow.on("closed", () => app.quit());

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url);
    return { action: "deny" };
  });
}

autoUpdater.checkForUpdatesAndNotify();

app.setName("Synnax");

app.whenReady().then(() => {
  createWindow();

  app.on("activate", function () {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });

  ipcMain.handle("getVersion", async () => {
    return app.getVersion();
  });

  const kvFilePath = join(app.getPath("userData"), "data.json");
  let data: Record<string, any> | null = null;

  const readKVData = async (): Promise<Record<string, any>> => {
    if (data !== null) return data;
    if (!fs.existsSync(kvFilePath)) data = {};
    else {
      try {
        const contents = await fs.promises.readFile(kvFilePath, "utf-8");
        data = JSON.parse(contents);
      } catch (e) {
        console.error(e);
        data = {};
      }
    }
    return data as Record<string, any>;
  };
  const mu = new Mutex();
  const writeKVData = async (d: Record<string, any>) => {
    await mu.runExclusive(async () => {
      await fs.promises.writeFile(kvFilePath, JSON.stringify(d, null, 2));
    });
  };

  ipcMain.handle("kvGet", async (_, key) => {
    const d = await readKVData();
    return d[key] ?? null;
  });
  ipcMain.handle("kvSet", async (_, key, value) => {
    const d = await readKVData();
    d[key] = value;
    await writeKVData(d);
  });
  ipcMain.handle("kvDelete", async (_, key) => {
    const d = await readKVData();
    delete d[key];
    await writeKVData(d);
  });
  app.on("before-quit", async () => {
    if (data !== null) await writeKVData(data);
  });
});
app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});
