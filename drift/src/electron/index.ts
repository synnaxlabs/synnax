import {
  BrowserViewConstructorOptions,
  BrowserWindowConstructorOptions,
  type BrowserWindow,
  type IpcRendererEvent,
} from "electron";
import type { Action, UnknownAction } from "@reduxjs/toolkit";
import { debounce, dimensions, type xy } from "@synnaxlabs/x";
import { MAIN_WINDOW, type WindowProps } from "@/window";
import { Event, Runtime } from "@/runtime";
import { SetWindowPropsPayload, setWindowProps } from "@/state";

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
const getInitialStateCmd = "drift://get-props";

const notFound = (key: string): Error => new Error(`Window not found: ${key}`);

export interface ListenOnMainProps {
  mainWindow: BrowserWindow;
  createWindow: (props: BrowserWindowConstructorOptions) => BrowserWindow;
}

const propsToOptions = (props: WindowProps): BrowserWindowConstructorOptions => ({
  x: props.position?.x,
  y: props.position?.y,
  width: props.size?.width,
  height: props.size?.height,
  center: props.center,
  minHeight: props.minSize?.height,
  minWidth: props.minSize?.width,
  maxHeight: props.maxSize?.height,
  maxWidth: props.maxSize?.width,
  resizable: props.resizable,
  fullscreen: props.fullscreen,
  skipTaskbar: props.skipTaskbar,
  title: props.title,
  show: props.visible,
  transparent: props.transparent,
  alwaysOnTop: props.alwaysOnTop,
});

const windowToProps = (win: BrowserWindow): Omit<WindowProps, "key"> => {
  const [width, height] = win.getSize();
  const [x, y] = win.getPosition();
  return {
    size: { width, height },
    position: { x, y },
    minimized: win.isMinimized(),
    maximized: win.isMaximized(),
    fullscreen: win.isFullScreen(),
    visible: win.isVisible(),
    resizable: win.isResizable(),
    skipTaskbar: false,
    title: win.getTitle(),
    alwaysOnTop: win.isAlwaysOnTop(),
    decorations: false,
  };
};

export const listenOnMain = ({ mainWindow, createWindow }: ListenOnMainProps) => {
  const labelsToIDs = new Map<string, number>();
  const idsToLabels = new Map<number, string>();
  labelsToIDs.set(MAIN_WINDOW, mainWindow.id);
  idsToLabels.set(mainWindow.id, MAIN_WINDOW);
  const { ipcMain, BrowserWindow } = require("electron");
  const listenOnSender = <A extends any>(
    ev: string,
    f: (win: BrowserWindow, args: A) => void,
  ) =>
    ipcMain.on(ev, (e: Electron.IpcMainEvent, args: A) => {
      const win = BrowserWindow.fromWebContents(e.sender);
      if (win == null) return;
      f(win, args);
    });
  ipcMain.on(closeEvent, (e: Electron.IpcMainEvent, label: string) => {
    const id = labelsToIDs.get(label);
    if (id == null) return;
    const win = BrowserWindow.fromId(id);
    if (win == null) return;
    win.close();
    labelsToIDs.delete(label);
    idsToLabels.delete(id);
  });
  ipcMain.handle(getInitialStateCmd, (e: Electron.IpcMainInvokeEvent) => {
    const win = BrowserWindow.fromWebContents(e.sender);
    if (win == null) return;
    const props = windowToProps(win);
    return { ...props, label: idsToLabels.get(win.id) };
  });

  const bindDebouncedHandler = (win: BrowserWindow, ev: string, f: () => void) => {
    win.on(ev as "will-resize", debounce(f, 500));
  };

  const sendToAll = (event: Event<any, any>) => {
    for (const id of idsToLabels.keys())
      BrowserWindow.fromId(id)?.webContents.send(actionEvent, event);
  };

  const bindHandlers = (label: string, win: BrowserWindow) => {
    const updateWindowPropsHandler = (
      ev: string,
      f: (win: BrowserWindow) => Omit<SetWindowPropsPayload, "label">,
    ) => {
      bindDebouncedHandler(win, ev, () => {
        sendToAll({
          action: setWindowProps({ label, ...f(win) }),
          emitter: "WHITELIST",
        });
      });
    };
    updateWindowPropsHandler("resize", (w) => {
      const [width, height] = w.getSize();
      return { size: { width, height } };
    });
    updateWindowPropsHandler("move", (w) => {
      const [x, y] = win.getPosition();
      return { position: { x, y } };
    });
    updateWindowPropsHandler("minimize", () => ({ minimized: true }));
    updateWindowPropsHandler("restore", () => ({ minimized: false }));
    updateWindowPropsHandler("maximize", () => ({ maximized: true }));
    updateWindowPropsHandler("unmaximize", () => ({ maximized: false }));
    updateWindowPropsHandler("enter-full-screen", (w) => {
      w.setWindowButtonVisibility(true);
      return { fullscreen: true };
    });
    updateWindowPropsHandler("leave-full-screen", () => {
      win.setWindowButtonVisibility(false);
      return { fullscreen: false };
    });
  };
  bindHandlers(MAIN_WINDOW, mainWindow);

  listenOnSender(focusEvent, (w) => w.focus());
  listenOnSender(setMinimizedEvent, (w) => w.minimize());
  listenOnSender(setMaximizedEvent, (w) => w.maximize());
  listenOnSender(setVisibleEvent, (w, value) => (value ? w.show() : w.hide()));
  listenOnSender(setFullscreenEvent, (w) => w.setFullScreen(true));
  listenOnSender(centerEvent, (w) => w.center());
  listenOnSender<xy.XY>(setPositionEvent, (w, { x, y }) => {
    w.setPosition(Math.round(x), Math.round(y));
  });
  listenOnSender<dimensions.Dimensions>(setSizeEvent, (w, { width, height }) => {
    w.setSize(Math.round(width), Math.round(height));
  });
  listenOnSender<dimensions.Dimensions>(setMinSizeEvent, (w, { width, height }) =>
    w.setMinimumSize(Math.round(width), Math.round(height)),
  );
  listenOnSender<dimensions.Dimensions>(setMaxSizeEvent, (w, { width, height }) =>
    w.setMaximumSize(Math.round(width), Math.round(height)),
  );
  listenOnSender<boolean>(setResizableEvent, (w, value) => w.setResizable(value));
  listenOnSender<boolean>(setSkipTaskbarEvent, (w, value) => w.setSkipTaskbar(value));
  listenOnSender<boolean>(setAlwaysOnTopEvent, (w, value) => w.setAlwaysOnTop(value));
  listenOnSender<string>(setTitleEvent, (w, title) => w.setTitle(title));
  ipcMain.on(
    createEvent,
    (e: Electron.IpcMainEvent, label: string, props: Omit<WindowProps, "key">) => {
      const win = createWindow({
        x: props.position?.x,
        y: props.position?.y,
        width: props.size?.width,
        height: props.size?.height,
        center: props.center,
        minHeight: props.minSize?.height,
        minWidth: props.minSize?.width,
        maxHeight: props.maxSize?.height,
        maxWidth: props.maxSize?.width,
        resizable: props.resizable,
        fullscreen: props.fullscreen,
        skipTaskbar: props.skipTaskbar,
        title: props.title,
        show: props.visible,
        transparent: props.transparent,
        alwaysOnTop: props.alwaysOnTop,
      });
      labelsToIDs.set(label, win.id);
      idsToLabels.set(win.id, label);
      bindHandlers(label, win);
    },
  );
  ipcMain.on(
    actionEvent,
    (e: Electron.IpcMainEvent, event: Event<any, any>, to?: string) => {
      if (event == null) return;
      try {
        if (to == null) {
          for (const [, id] of labelsToIDs.entries()) {
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
      } catch (e) {
        console.error(e);
      }
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
  private props: Omit<WindowProps, "key"> | null = null;

  constructor() {
    this.unsubscribe = {};
  }

  async configure(): Promise<void> {
    const { label, ...props } = await window.electronAPI.invoke(getInitialStateCmd);
    this._label = label;
    this.props = props;
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
    window.electronAPI.send(createEvent, label, JSON.parse(JSON.stringify(props)));
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
    if (this.props != null) return this.props;
    throw new Error("Window not found");
  }
}
