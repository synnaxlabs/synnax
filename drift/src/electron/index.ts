import { type BrowserWindow, type IpcRendererEvent } from "electron";
import type { Action, UnknownAction } from "@reduxjs/toolkit";
import { type dimensions, type xy } from "@synnaxlabs/x";
import { MAIN_WINDOW, type WindowProps } from "@/window";
import { Event, Runtime } from "@/runtime";

const actionEvent = "drift://action";
const createEvent = "drift://create";
const focusEvent = "drift://focus";
const closeEvent = "drift://close";
const setMinimizedEvent = "drift://set-minimized";
const setMaximizedEvent = "drift://set-maximized";
const setVisibleEvent = "drift://set-visible";
const setFullscreenEvent = "drift://set-fullscreen";
const centerEvent = "drift://center";
const setPositionEvent = "drift://set-position";
const setSizeEvent = "drift://set-size";
const setMinSizeEvent = "drift://set-min-size";
const setMaxSizeEvent = "drift://set-max-size";
const setResizableEvent = "drift://set-resizable";
const setSkipTaskbarEvent = "drift://set-skip-taskbar";
const setAlwaysOnTopEvent = "drift://set-always-on-top";
const setTitleEvent = "drift://set-title";
const setDecorationsEvent = "drift://set-decorations";
const getLabelCmd = "drift://get-label";

const notFound = (key: string): Error => new Error(`Window not found: ${key}`);

export interface ListenOnMainProps {
  mainWindow: BrowserWindow;
  createWindow: (props: Omit<WindowProps, "key">) => BrowserWindow;
}

export const listenOnMain = ({ mainWindow, createWindow }: ListenOnMainProps) => {
  const labelsToIDs = new Map<string, number>();
  const idsToLabels = new Map<number, string>();
  labelsToIDs.set(MAIN_WINDOW, mainWindow.id);
  idsToLabels.set(mainWindow.id, MAIN_WINDOW);
  const { ipcMain, BrowserWindow } = require("electron");
  const exec =
    <A extends any>(f: (win: BrowserWindow, args: A) => void) =>
    (e: Electron.IpcMainEvent, args: A) => {
      const win = BrowserWindow.fromWebContents(e.sender);
      if (win == null) return;
      f(win, args);
    };
  ipcMain.handle(getLabelCmd, (e: Electron.IpcMainInvokeEvent) => {
    const win = BrowserWindow.fromWebContents(e.sender);
    if (win == null) return;
    return idsToLabels.get(win.id);
  });
  ipcMain.on(
    closeEvent,
    exec((w) => w.close()),
  );
  ipcMain.on(
    focusEvent,
    exec((w) => w.focus()),
  );
  ipcMain.on(
    setMinimizedEvent,
    exec((w) => w.minimize()),
  );
  ipcMain.on(
    setMaximizedEvent,
    exec((w) => w.maximize()),
  );
  ipcMain.on(
    setVisibleEvent,
    exec((w, value) => (value ? w.show() : w.hide())),
  );
  ipcMain.on(
    setFullscreenEvent,
    exec((w) => w.setFullScreen(true)),
  );
  ipcMain.on(
    centerEvent,
    exec((w) => w.center()),
  );
  ipcMain.on(
    setPositionEvent,
    exec<xy.XY>((w, { x, y }) => w.setPosition(x, y)),
  );
  ipcMain.on(
    setSizeEvent,
    exec<dimensions.Dimensions>((w, { width, height }) => w.setSize(width, height)),
  );
  ipcMain.on(
    setMinSizeEvent,
    exec<dimensions.Dimensions>((w, { width, height }) =>
      w.setMinimumSize(width, height),
    ),
  );
  ipcMain.on(
    setMaxSizeEvent,
    exec<dimensions.Dimensions>((w, { width, height }) =>
      w.setMaximumSize(width, height),
    ),
  );
  ipcMain.on(
    setResizableEvent,
    exec<boolean>((w, value) => w.setResizable(value)),
  );
  ipcMain.on(
    setSkipTaskbarEvent,
    exec<boolean>((w, value) => w.setSkipTaskbar(value)),
  );
  ipcMain.on(
    setAlwaysOnTopEvent,
    exec<boolean>((w, value) => w.setAlwaysOnTop(value)),
  );
  ipcMain.on(
    setTitleEvent,
    exec<string>((w, title) => w.setTitle(title)),
  );
  ipcMain.on(
    createEvent,
    (e: Electron.IpcMainEvent, label: string, props: Omit<WindowProps, "key">) => {
      const win = createWindow(props);
      labelsToIDs.set(label, win.id);
      idsToLabels.set(win.id, label);
    },
  );
  ipcMain.on(
    actionEvent,
    (e: Electron.IpcMainEvent, event: Event<any, any>, to?: string) => {
      if (event == null) return;
      if (to == null) {
        // get all windows
        for (const [label, id] of labelsToIDs.entries()) {
          const win = BrowserWindow.fromId(id);
          if (win == null) return;
          win.webContents.send(actionEvent, event);
        }
        return;
      }
      const id = labelsToIDs.get(to);
      if (id == null) return;
      const win = BrowserWindow.fromId(id);
      if (win == null) return;
      win.webContents.send(actionEvent, event);
    },
  );
};

export const configurePreload = () => {
  const { ipcRenderer, contextBridge } = require("electron");
  contextBridge.exposeInMainWorld("electronAPI", {
    send: (channel: string, ...args: any[]) => ipcRenderer.send(channel, ...args),
    invoke: (cmd: string, ...args: any[]) => ipcRenderer.invoke(cmd, ...args),
    on: (event: string, listener: (event: IpcRendererEvent, ...args: any[]) => void) =>
      ipcRenderer.on(event, listener),
  });
};

/**
 * An Electron backed implementation of the drift Runtime.
 */
export class ElectronRuntime<S extends StoreState, A extends Action = UnknownAction>
  implements Runtime<S, A>
{
  private unsubscribe: Record<string, () => void>;
  private fullscreenPoll: any | null = null;
  private subscribeCallback: ((action: Event<S, A>) => void) | null = null;
  private _label: string = "";

  constructor() {
    this.unsubscribe = {};
  }

  async configure(): Promise<void> {
    this._label = await window.electronAPI.invoke(getLabelCmd);
  }

  label(): string {
    return this._label;
  }

  isMain(): boolean {
    return this._label === MAIN_WINDOW;
  }

  release(): void {
    Object.values(this.unsubscribe).forEach((f) => f?.());
    this.unsubscribe = {};
  }

  async emit(event_: Omit<Event<S, A>, "emitter">, to?: string): Promise<void> {
    window.electronAPI.send(actionEvent, { ...event_, emitter: this.label() }, to);
  }

  async subscribe(lis: (action: Event<S, A>) => void): Promise<void> {
    window.electronAPI.on(actionEvent, (_, event: Event<S, A>) => lis(event));
  }

  onCloseRequested(cb: () => void): void {}

  async create(label: string, props: Omit<WindowProps, "key">): Promise<void> {
    window.electronAPI.send(createEvent, label, props);
  }

  async close(key: string): Promise<void> {
    window.electronAPI.send(closeEvent, key);
  }

  listLabels(): string[] {
    return [];
  }

  async focus(): Promise<void> {
    window.electronAPI.send(focusEvent);
  }

  async setMinimized(value: boolean): Promise<void> {
    window.electronAPI.send(setMinimizedEvent, value);
  }

  async setMaximized(value: boolean): Promise<void> {
    window.electronAPI.send(setMaximizedEvent, value);
  }

  async setVisible(value: boolean): Promise<void> {
    window.electronAPI.send(setVisibleEvent, value);
  }

  async setFullscreen(value: boolean): Promise<void> {
    window.electronAPI.send(setFullscreenEvent, value);
  }

  async center(): Promise<void> {
    window.electronAPI.send(centerEvent);
  }

  async setPosition(xy: xy.XY): Promise<void> {
    window.electronAPI.send(setPositionEvent, xy);
  }

  async setSize(dims: dimensions.Dimensions): Promise<void> {
    window.electronAPI.send(setSizeEvent, dims);
  }

  async setMinSize(dims: dimensions.Dimensions): Promise<void> {
    window.electronAPI.send(setMinSizeEvent, dims);
  }

  async setMaxSize(dims: dimensions.Dimensions): Promise<void> {
    window.electronAPI.send(setMaxSizeEvent, dims);
  }

  async setResizable(value: boolean): Promise<void> {
    window.electronAPI.send(setResizableEvent, value);
  }

  async setSkipTaskbar(value: boolean): Promise<void> {
    window.electronAPI.send(setSkipTaskbarEvent, value);
  }

  async setAlwaysOnTop(value: boolean): Promise<void> {
    window.electronAPI.send(setAlwaysOnTopEvent, value);
  }

  async setTitle(title: string): Promise<void> {
    window.electronAPI.send(setTitleEvent, title);
  }

  async setDecorations(value: boolean): Promise<void> {
    window.electronAPI.send(setDecorationsEvent, value);
  }

  async getProps(): Promise<Omit<WindowProps, "key">> {
    return {
      size: { width: 0, height: 0 },
      minSize: { width: 0, height: 0 },
      maxSize: { width: 0, height: 0 },
      position: { x: 0, y: 0 },
      fullscreen: false,
      visible: false,
      resizable: false,
      skipTaskbar: false,
      alwaysOnTop: false,
      title: "",
      decorations: false,
    };
  }
}
