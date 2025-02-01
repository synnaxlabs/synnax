// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";
import { debounce, type dimensions, type xy } from "@synnaxlabs/x";
import {
  type BrowserWindow,
  type BrowserWindowConstructorOptions,
  type IpcMainEvent,
  type IpcRendererEvent,
} from "electron";

import { type Event, type Runtime } from "@/runtime";
import {
  runtimeSetWindowProps,
  type RuntimeSetWindowProsPayload,
  type StoreState,
} from "@/state";
import { MAIN_WINDOW, type WindowProps } from "@/window";

const ACTION_EVENT = "drift://action";
const CREATE_EVENT = "drift://create";
const FOCUS_EVENT = "drift://focus";
const CLOSE_EVENT = "drift://close";
const SET_MINIMIZED_EVENT = "drift://set-minimized";
const SET_MAXIMIZED_EVENT = "drift://set-maximized";
const SET_VISIBLE_EVENT = "drift://set-visible";
const SET_FULLSCREEN_EVENT = "drift://set-fullscreen";
const CENTER_EVENT = "drift://center";
const SET_POSITION_EVENT = "drift://set-position";
const SET_SIZE_EVENT = "drift://set-size";
const SET_MIN_SIZE_EVENT = "drift://set-min-size";
const SET_MAX_SIZE_EVENT = "drift://set-max-size";
const SET_RESIZABLE_EVENT = "drift://set-resizable";
const SET_SKIP_TASKBAR_EVENT = "drift://set-skip-taskbar";
const SET_ALWAYS_ON_TOP_EVENT = "drift://set-always-on-top";
const SET_TITLE_EVENT = "drift://set-title";
const SET_DECORATIONS_EVENT = "drift://set-decorations";
const GET_INITIAL_STATE_CMD = "drift://get-props";
const GET_WINDOW_LABEL_CMD = "drift://get-label";

const OUTBOUND_EVENTS = [
  ACTION_EVENT,
  CREATE_EVENT,
  FOCUS_EVENT,
  CLOSE_EVENT,
  SET_MINIMIZED_EVENT,
  SET_MAXIMIZED_EVENT,
  SET_VISIBLE_EVENT,
  SET_FULLSCREEN_EVENT,
  CENTER_EVENT,
  SET_POSITION_EVENT,
  SET_SIZE_EVENT,
  SET_MIN_SIZE_EVENT,
  SET_MAX_SIZE_EVENT,
  SET_RESIZABLE_EVENT,
  SET_SKIP_TASKBAR_EVENT,
  SET_ALWAYS_ON_TOP_EVENT,
  SET_TITLE_EVENT,
  SET_DECORATIONS_EVENT,
  GET_INITIAL_STATE_CMD,
];
type OutboundEvent = (typeof OUTBOUND_EVENTS)[number];
const validateOutboundEvent = (event: string): void => {
  if (!OUTBOUND_EVENTS.includes(event))
    throw new Error(`Event ${event} is not on the list of allowed events`);
};

const INBOUND_EVENTS = [ACTION_EVENT];
type InBoundEvent = (typeof INBOUND_EVENTS)[number];
const validateInboundEvent = (event: string): void => {
  if (!INBOUND_EVENTS.includes(event))
    throw new Error(`Event ${event} is not on the list of allowed events`);
};

const VALID_COMMANDS = [GET_INITIAL_STATE_CMD, GET_WINDOW_LABEL_CMD];
type Command = (typeof VALID_COMMANDS)[number];
const validateCommand = (cmd: string): void => {
  if (!VALID_COMMANDS.includes(cmd))
    throw new Error(`Command ${cmd} is not on the list of allowed commands`);
};

export interface ListenOnMainProps {
  mainWindow: BrowserWindow;
  createWindow: (props: BrowserWindowConstructorOptions) => BrowserWindow;
}

const propsToOptions = (
  props: Omit<WindowProps, "key">,
): BrowserWindowConstructorOptions => ({
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
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { ipcMain, BrowserWindow } = require("electron");
  const listenOnSender = <A>(ev: string, f: (win: BrowserWindow, args: A) => void) =>
    ipcMain.on(ev, (e: IpcMainEvent, args: A) => {
      const win = BrowserWindow.fromWebContents(e.sender);
      if (win == null) return;
      f(win, args);
    });
  ipcMain.on(CLOSE_EVENT, (_: IpcMainEvent, label: string) => {
    const id = labelsToIDs.get(label);
    if (id == null) return;
    const win = BrowserWindow.fromId(id);
    if (win == null) return;
    win.close();
    labelsToIDs.delete(label);
    idsToLabels.delete(id);
  });
  ipcMain.handle(GET_INITIAL_STATE_CMD, (e: Electron.IpcMainInvokeEvent) => {
    const win = BrowserWindow.fromWebContents(e.sender);
    if (win == null) return;
    const props = windowToProps(win);
    return { ...props, label: idsToLabels.get(win.id) };
  });
  ipcMain.handle(GET_WINDOW_LABEL_CMD, (e: Electron.IpcMainInvokeEvent) => {
    const win = BrowserWindow.fromWebContents(e.sender);
    if (win == null) return;
    return idsToLabels.get(win.id);
  });

  const bindDebouncedHandler = (win: BrowserWindow, ev: string, f: () => void) => {
    win.on(ev as "will-resize", debounce(f, 500));
  };

  const sendToAll = (event: Event<any, any>) => {
    for (const id of idsToLabels.keys())
      BrowserWindow.fromId(id)?.webContents.send(ACTION_EVENT, event);
  };

  const bindHandlers = (label: string, win: BrowserWindow) => {
    const updateWindowPropsHandler = (
      ev: string,
      f: (win: BrowserWindow) => Omit<RuntimeSetWindowProsPayload, "label">,
    ) => {
      bindDebouncedHandler(win, ev, () => {
        sendToAll({
          action: runtimeSetWindowProps({ label, ...f(win) }),
          emitter: "WHITELIST",
        });
      });
    };
    updateWindowPropsHandler("resize", (w) => {
      const [width, height] = w.getSize();
      return { size: { width, height } };
    });
    updateWindowPropsHandler("move", () => {
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

  listenOnSender(FOCUS_EVENT, (w) => w.focus());
  listenOnSender(SET_MINIMIZED_EVENT, (w) => w.minimize());
  listenOnSender(SET_MAXIMIZED_EVENT, (w) => w.maximize());
  listenOnSender(SET_VISIBLE_EVENT, (w, value) => (value ? w.show() : w.hide()));
  listenOnSender(SET_FULLSCREEN_EVENT, (w) => w.setFullScreen(true));
  listenOnSender(CENTER_EVENT, (w) => w.center());
  listenOnSender<xy.XY>(SET_POSITION_EVENT, (w, { x, y }) => {
    w.setPosition(Math.round(x), Math.round(y));
  });
  listenOnSender<dimensions.Dimensions>(SET_SIZE_EVENT, (w, { width, height }) => {
    w.setSize(Math.round(width), Math.round(height));
  });
  listenOnSender<dimensions.Dimensions>(SET_MIN_SIZE_EVENT, (w, { width, height }) =>
    w.setMinimumSize(Math.round(width), Math.round(height)),
  );
  listenOnSender<dimensions.Dimensions>(SET_MAX_SIZE_EVENT, (w, { width, height }) =>
    w.setMaximumSize(Math.round(width), Math.round(height)),
  );
  listenOnSender<boolean>(SET_RESIZABLE_EVENT, (w, value) => w.setResizable(value));
  listenOnSender<boolean>(SET_SKIP_TASKBAR_EVENT, (w, value) =>
    w.setSkipTaskbar(value),
  );
  listenOnSender<boolean>(SET_ALWAYS_ON_TOP_EVENT, (w, value) =>
    w.setAlwaysOnTop(value),
  );
  listenOnSender<string>(SET_TITLE_EVENT, (w, title) => w.setTitle(title));
  ipcMain.on(
    CREATE_EVENT,
    (_: IpcMainEvent, label: string, props: Omit<WindowProps, "key">) => {
      const win = createWindow(propsToOptions(props));
      labelsToIDs.set(label, win.id);
      idsToLabels.set(win.id, label);
      bindHandlers(label, win);
    },
  );
  ipcMain.on(ACTION_EVENT, (_: IpcMainEvent, event: Event<any, any>, to?: string) => {
    if (event == null) return;
    if (to == null) return sendToAll(event);
    const id = labelsToIDs.get(to);
    if (id == null) return;
    const win = BrowserWindow.fromId(id);
    if (win == null) return;
    win.webContents.send(ACTION_EVENT, event);
  });
};

interface API {
  send: (channel: OutboundEvent, ...args: any[]) => void;
  invoke: (cmd: Command, ...args: any[]) => Promise<any>;
  on: (
    event: InBoundEvent,
    listener: (event: IpcRendererEvent, ...args: any[]) => void,
  ) => void;
}

const API_NAME = "driftAPI";

export const exposeAPI = () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { ipcRenderer, contextBridge } = require("electron");
  const api: API = {
    send: (channel, ...args) => {
      validateOutboundEvent(channel);
      ipcRenderer.send(channel, ...args);
    },
    invoke: async (cmd, ...args) => {
      validateCommand(cmd);
      return await ipcRenderer.invoke(cmd, ...args);
    },
    on: (event, listener) => {
      validateInboundEvent(event);
      ipcRenderer.on(event, listener);
    },
  };
  contextBridge.exposeInMainWorld(API_NAME, api);
};

export const getWindowLabel = async (): Promise<string> => {
  if (!(API_NAME in window))
    throw new Error(
      "Drift API not found. Make sure to call configurePreload in your preload script.",
    );
  return await (window as { [API_NAME]: API })[API_NAME].invoke(GET_WINDOW_LABEL_CMD);
};

/**
 * An Electron backed implementation of the drift Runtime.
 */
export class ElectronRuntime<S extends StoreState, A extends Action = UnknownAction>
  implements Runtime<S, A>
{
  private _label: string = "";
  private props: Omit<WindowProps, "key"> | null = null;
  private api: API;

  constructor() {
    if (!(API_NAME in window))
      throw new Error(
        "Drift API not found. Make sure to call configurePreload in your preload script.",
      );
    this.api = window[API_NAME] as API;
  }

  async configure(): Promise<void> {
    const { label, ...props } = await this.api.invoke(GET_INITIAL_STATE_CMD);
    this._label = label;
    this.props = props;
  }

  label(): string {
    return this._label;
  }

  isMain(): boolean {
    return this._label === MAIN_WINDOW;
  }

  release(): void {}

  async emit(event_: Omit<Event<S, A>, "emitter">, to?: string): Promise<void> {
    this.api.send(ACTION_EVENT, { ...event_, emitter: this.label() }, to);
  }

  async subscribe(lis: (action: Event<S, A>) => void): Promise<void> {
    this.api.on(ACTION_EVENT, (_, event: Event<S, A>) => lis(event));
  }

  onCloseRequested(): void {}

  async create(label: string, props: Omit<WindowProps, "key">): Promise<void> {
    this.api.send(CREATE_EVENT, label, JSON.parse(JSON.stringify(props)));
  }

  async close(key: string): Promise<void> {
    this.api.send(CLOSE_EVENT, key);
  }

  async listLabels(): Promise<string[]> {
    return [];
  }

  async focus(): Promise<void> {
    this.api.send(FOCUS_EVENT);
  }

  async setMinimized(value: boolean): Promise<void> {
    this.api.send(SET_MINIMIZED_EVENT, value);
  }

  async setMaximized(value: boolean): Promise<void> {
    this.api.send(SET_MAXIMIZED_EVENT, value);
  }

  async setVisible(value: boolean): Promise<void> {
    this.api.send(SET_VISIBLE_EVENT, value);
  }

  async setFullscreen(value: boolean): Promise<void> {
    this.api.send(SET_FULLSCREEN_EVENT, value);
  }

  async center(): Promise<void> {
    this.api.send(CENTER_EVENT);
  }

  async setPosition(xy: xy.XY): Promise<void> {
    this.api.send(SET_POSITION_EVENT, xy);
  }

  async setSize(dims: dimensions.Dimensions): Promise<void> {
    this.api.send(SET_SIZE_EVENT, dims);
  }

  async setMinSize(dims: dimensions.Dimensions): Promise<void> {
    this.api.send(SET_MIN_SIZE_EVENT, dims);
  }

  async setMaxSize(dims: dimensions.Dimensions): Promise<void> {
    this.api.send(SET_MAX_SIZE_EVENT, dims);
  }

  async setResizable(value: boolean): Promise<void> {
    this.api.send(SET_RESIZABLE_EVENT, value);
  }

  async setSkipTaskbar(value: boolean): Promise<void> {
    this.api.send(SET_SKIP_TASKBAR_EVENT, value);
  }

  async setAlwaysOnTop(value: boolean): Promise<void> {
    this.api.send(SET_ALWAYS_ON_TOP_EVENT, value);
  }

  async setTitle(title: string): Promise<void> {
    this.api.send(SET_TITLE_EVENT, title);
  }

  async setDecorations(value: boolean): Promise<void> {
    this.api.send(SET_DECORATIONS_EVENT, value);
  }

  async getProps(): Promise<Omit<WindowProps, "key">> {
    if (this.props != null) return this.props;
    throw new Error("Window not found");
  }
}
