// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, AnyAction } from "@reduxjs/toolkit";
import { debounce as debounceF, Dimensions, XY } from "@synnaxlabs/x";
import type { Event as TauriEvent, UnlistenFn } from "@tauri-apps/api/event";
import { listen, emit, TauriEvent as TauriEventKey } from "@tauri-apps/api/event";
import {
  WebviewWindow,
  appWindow,
  LogicalPosition,
  LogicalSize,
} from "@tauri-apps/api/window";

import { Event, Runtime } from "@/runtime";
import { decode, encode } from "@/serialization";
import { setWindowProps, StoreState } from "@/state";
import { KeyedWindowProps, MAIN_WINDOW } from "@/window";

const actionEvent = "drift://action";
const tauriError = "tauri://error";
const notFound = (key: string): Error => new Error(`Window not found: ${key}`);

/**
 * A Tauri backed implementation of the drift Runtime.
 */
export class TauriRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  private readonly win: WebviewWindow;
  private readonly unsubscribe: UnlistenFn[];

  /**
   * @param window - The WebviewWindow to use as the underlying engine for this runtime.
   * This should not be set in 99% of cases. Only use this if you know what you're doing.
   */
  constructor(window?: WebviewWindow) {
    this.win = window ?? appWindow;
    this.unsubscribe = [];
  }

  key(): string {
    return this.win.label;
  }

  isMain(): boolean {
    return this.win.label === MAIN_WINDOW;
  }

  release(): void {
    this.unsubscribe.forEach((f) => f?.());
  }

  emit(event_: Omit<Event<S, A>, "emitter">, to?: string): void {
    const event = encode({ ...event_, emitter: this.key() });
    if (to != null) {
      const win = WebviewWindow.getByLabel(to);
      if (win == null) throw notFound(to);
      void win.emit(actionEvent, event);
    } else {
      void emit(actionEvent, event);
    }
  }

  subscribe(lis: (action: Event<S, A>) => void): void {
    void listen<string>(actionEvent, (event: TauriEvent<string>) => {
      lis(decode(event.payload));
    })
      .catch(console.error)
      .then((unlisten) => {
        if (unlisten != null) this.unsubscribe.push(unlisten);
      });

    newEventHandlers().forEach(({ key, handler, debounce }) => {
      void listen(
        key,
        debounceF((event: TauriEvent<any>) => {
          if (event.windowLabel !== this.key()) return;
          void handler(event).then((action) => {
            if (action != null) lis({ action: action as A, emitter: "WHITELIST" });
          });
        }, debounce)
      ).then((unlisten) => {
        if (unlisten != null) this.unsubscribe.push(unlisten);
      });
    });
  }

  onCloseRequested(cb: () => void): void {
    void this.win.onCloseRequested((e) => {
      // Only propagate the close request if the event
      // is for the current window.
      if (e.windowLabel !== this.key()) return;
      // Prevent default so the window doesn't close
      // until all processes are complete.
      e.preventDefault();
      cb();
    });
  }

  // |||||| MANAGER IMPLEMENTATION ||||||

  create({ key, ...props }: KeyedWindowProps): void {
    const w = new WebviewWindow(key, {
      ...props,
      visible: false,
    });
    void w.once(tauriError, console.error);
  }

  async close(key: string): Promise<void> {
    const win = WebviewWindow.getByLabel(key);
    if (win != null) await win.close();
  }

  async focus(): Promise<void> {
    return await this.win.setFocus();
  }

  async setMinimized(value: boolean): Promise<void> {
    return value ? await this.win.minimize() : await this.win.unminimize();
  }

  async setMaximized(value: boolean): Promise<void> {
    return value ? await this.win.maximize() : await this.win.unmaximize();
  }

  async setVisible(value: boolean): Promise<void> {
    return value ? await this.win.show() : await this.win.hide();
  }

  async setFullscreen(value: boolean): Promise<void> {
    return await this.win.setFullscreen(value);
  }

  async center(): Promise<void> {
    return await this.win.center();
  }

  async setPosition(xy: XY): Promise<void> {
    void this.win.setPosition(new LogicalPosition(xy.x, xy.y));
  }

  async setSize(dims: Dimensions): Promise<void> {
    void this.win.setSize(new LogicalSize(dims.width, dims.height));
  }

  async setMinSize(dims: Dimensions): Promise<void> {
    void this.win.setMinSize(new LogicalSize(dims.width, dims.height));
  }

  async setMaxSize(dims: Dimensions): Promise<void> {
    void this.win.setMaxSize(new LogicalSize(dims.width, dims.height));
  }

  async setResizable(value: boolean): Promise<void> {
    return await this.win.setResizable(value);
  }

  async setSkipTaskbar(value: boolean): Promise<void> {
    return await this.win.setSkipTaskbar(value);
  }

  async setAlwaysOnTop(value: boolean): Promise<void> {
    return await this.win.setAlwaysOnTop(value);
  }

  async setTitle(title: string): Promise<void> {
    return await this.win.setTitle(title);
  }
}

interface HandlerEntry {
  key: TauriEventKey;
  debounce: number;
  handler: (ev: TauriEvent<any>) => Promise<AnyAction>;
}

const newEventHandlers = (): HandlerEntry[] => [
  {
    key: TauriEventKey.WINDOW_RESIZED,
    debounce: 500,
    handler: async (ev) => {
      const window = WebviewWindow.getByLabel(ev.windowLabel);
      const pos = await window?.innerPosition();
      const dims = await window?.innerSize();
      const maximized = await window?.isMaximized();
      const visible = await window?.isVisible();
      return setWindowProps({
        x: pos?.x,
        y: pos?.y,
        height: dims?.height,
        width: dims?.width,
        maximized,
        visible,
        key: ev.windowLabel,
      });
    },
  },
  {
    key: TauriEventKey.WINDOW_MOVED,
    debounce: 1000,
    handler: async (ev) => {
      const window = WebviewWindow.getByLabel(ev.windowLabel);
      const pos = await window?.innerPosition();
      // wait 5000 ms
      const fullscreen = await window?.isFullscreen();
      return setWindowProps({ x: pos?.x, y: pos?.y, fullscreen, key: ev.windowLabel });
    },
  },
  {
    key: TauriEventKey.WINDOW_BLUR,
    debounce: 0,
    handler: async (ev) => setWindowProps({ focus: false, key: ev.windowLabel }),
  },
  {
    key: TauriEventKey.WINDOW_FOCUS,
    debounce: 0,
    handler: async (ev) => setWindowProps({ focus: true, key: ev.windowLabel }),
  },
];
