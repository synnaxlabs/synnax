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
  getAll,
} from "@tauri-apps/api/window";

import { Event, Runtime } from "@/runtime";
import { decode, encode } from "@/serialization";
import { setWindowProps, StoreState } from "@/state";
import { LabeledWindowProps, MAIN_WINDOW } from "@/window";

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
  private readonly unsubscribe: Record<string, UnlistenFn>;

  /**
   * @param window - The WebviewWindow to use as the underlying engine for this runtime.
   * This should not be set in 99% of cases. Only use this if you know what you're doing.
   */
  constructor(window?: WebviewWindow) {
    this.win = window ?? appWindow;
    this.unsubscribe = {};
  }

  label(): string {
    return this.win.label;
  }

  isMain(): boolean {
    return this.win.label === MAIN_WINDOW;
  }

  release(): void {
    Object.values(this.unsubscribe).forEach((f) => f?.());
  }

  emit(event_: Omit<Event<S, A>, "emitter">, to?: string): void {
    const event = encode({ ...event_, emitter: this.label() });
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
        if (unlisten != null) this.unsubscribe[actionEvent] = unlisten;
      });

    newEventHandlers().forEach(({ key, handler, debounce, condition }) => {
      const lis_ = async (): Promise<void> =>
        await this.win
          .listen(
            key,
            debounceF((event: TauriEvent<any>) => {
              if (event.windowLabel !== this.label()) return;
              void handler(event).then((action) => {
                if (action != null) lis({ action: action as A, emitter: "WHITELIST" });
              });
            }, debounce)
          )
          .then((unlisten) => {
            if (unlisten != null) {
              this.unsubscribe[key] = unlisten;
            }
          });
      if (condition != null)
        void condition(this.win).then(async (ok) => await (ok && lis_()));
      else void lis_();
    });
  }

  onCloseRequested(cb: () => void): void {
    void this.win.onCloseRequested((e) => {
      // Only propagate the close request if the event
      // is for the current window.
      if (e.windowLabel !== this.label()) return;
      // Prevent default so the window doesn't close
      // until all processes are complete.
      e.preventDefault();
      cb();
    });
  }

  // |||||| MANAGER IMPLEMENTATION ||||||

  create({ label, ...props }: LabeledWindowProps): void {
    const { size, minSize, maxSize, position, ...rest } = props;
    const w = new WebviewWindow(label, {
      x: position?.x,
      y: position?.y,
      width: size?.width,
      height: size?.height,
      minWidth: minSize?.width,
      minHeight: minSize?.height,
      maxWidth: maxSize?.width,
      maxHeight: maxSize?.height,
      ...rest,
    });
    void w.once(tauriError, console.error);
  }

  async close(key: string): Promise<void> {
    const win = WebviewWindow.getByLabel(key);
    if (win != null) await win.close();
  }

  listLabels(): string[] {
    return getAll().map((w) => w.label);
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
    if (TauriEventKey.WINDOW_RESIZED in this.unsubscribe && !value) {
      void this.unsubscribe[TauriEventKey.WINDOW_RESIZED]?.();
      delete this.unsubscribe[TauriEventKey.WINDOW_RESIZED];
    }
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
  condition?: (win: WebviewWindow | null) => Promise<boolean>;
  handler: (ev: TauriEvent<any>) => Promise<AnyAction | null>;
}

const newEventHandlers = (): HandlerEntry[] => [
  {
    key: TauriEventKey.WINDOW_RESIZED,
    debounce: 200,
    handler: async (ev) => {
      const window = WebviewWindow.getByLabel(ev.windowLabel);
      const scaleFactor = await window?.scaleFactor();
      if (scaleFactor == null) return null;
      const position = (await window?.innerPosition())?.toLogical(scaleFactor);
      const size = (await window?.innerSize())?.toLogical(scaleFactor);
      const maximized = await window?.isMaximized();
      const visible = await window?.isVisible();
      return setWindowProps({
        // We need to do it this way or else we'll put non-serializable values into the store
        position: { x: position?.x ?? 0, y: position?.y ?? 0 },
        size: { width: size?.width ?? 0, height: size?.height ?? 0 },
        maximized,
        visible,
        minimized: !(visible ?? false),
        label: ev.windowLabel,
      });
    },
  },
  {
    key: TauriEventKey.WINDOW_MOVED,
    debounce: 200,
    handler: async (ev) => {
      const window = WebviewWindow.getByLabel(ev.windowLabel);
      const scaleFactor = await window?.scaleFactor();
      if (scaleFactor == null) return null;
      const position = (await window?.innerPosition())?.toLogical(scaleFactor);
      const fullscreen = await window?.isFullscreen();
      const size = (await window?.innerSize())?.toLogical(scaleFactor);
      const visible = await window?.isVisible();
      return setWindowProps({
        label: ev.windowLabel,
        position: { x: position?.x ?? 0, y: position?.y ?? 0 },
        fullscreen,
        size: { width: size?.width ?? 0, height: size?.height ?? 0 },
        visible,
      });
    },
  },
  {
    key: TauriEventKey.WINDOW_BLUR,
    debounce: 0,
    handler: async (ev) => setWindowProps({ focus: false, label: ev.windowLabel }),
  },
  {
    key: TauriEventKey.WINDOW_FOCUS,
    debounce: 0,
    handler: async (ev) =>
      setWindowProps({
        focus: true,
        visible: true,
        minimized: false,
        label: ev.windowLabel,
      }),
  },
];
