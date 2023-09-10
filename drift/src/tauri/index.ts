// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, AnyAction } from "@reduxjs/toolkit";
import { debounce as debounceF, dimensions, xy } from "@synnaxlabs/x";
import type { Event as TauriEvent, UnlistenFn } from "@tauri-apps/api/event";
import { listen, emit, TauriEvent as TauriEventKey } from "@tauri-apps/api/event";
import {
  WebviewWindow,
  appWindow,
  LogicalPosition,
  LogicalSize,
  getAll,
  PhysicalPosition,
  PhysicalSize,
} from "@tauri-apps/api/window";

import { Event, Runtime } from "@/runtime";
import { decode, encode } from "@/serialization";
import { setWindowProps, SetWindowPropsPayload, StoreState } from "@/state";
import { MAIN_WINDOW, WindowProps } from "@/window";

const actionEvent = "drift://action";
const tauriError = "tauri://error";
const tauriCreated = "tauri://created";
const notFound = (key: string): Error => new Error(`Window not found: ${key}`);

/**
 * A Tauri backed implementation of the drift Runtime.
 */
export class TauriRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  private readonly win: WebviewWindow;
  private unsubscribe: Record<string, UnlistenFn>;
  private fullscreenPoll: any | null = null;
  private subscribeCallback: ((action: Event<S, A>) => void) | null = null;

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
    this.unsubscribe = {};
  }

  async emit(event_: Omit<Event<S, A>, "emitter">, to?: string): Promise<void> {
    const event = encode({ ...event_, emitter: this.label() });
    if (to == null) return await emit(actionEvent, event);
    const win = WebviewWindow.getByLabel(to);
    if (win == null) throw notFound(to);
    await win.emit(actionEvent, event);
  }

  async subscribe(lis: (action: Event<S, A>) => void): Promise<void> {
    this.subscribeCallback = lis;
    this.release();
    this.unsubscribe[actionEvent] = await listen<string>(
      actionEvent,
      (event: TauriEvent<string>) => lis(decode(event.payload))
    );
    const propsHandlers = newWindowPropsHandlers();
    for (const { key, handler, debounce } of propsHandlers) {
      this.unsubscribe[key] = await this.win.listen(
        key,
        debounceF((event: TauriEvent<any>) => {
          if (event.windowLabel !== this.label()) return;
          void handler(event).then((action) => {
            if (action != null) lis({ action: action as A, emitter: "WHITELIST" });
          });
        }, debounce)
      );
    }
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

  async create(label: string, props: Omit<WindowProps, "key">): Promise<void> {
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
    return await new Promise<void>((resolve, reject) => {
      void w.once(tauriError, (e) => reject(e.payload));
      void w.once(tauriCreated, () => resolve());
    });
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
    if (this.fullscreenPoll != null) clearInterval(this.fullscreenPoll);
    if (value)
      this.fullscreenPoll = setInterval(() => {
        this.win
          .isFullscreen()
          .then((isFullscreen) => {
            if (!isFullscreen) {
              this.subscribeCallback?.({
                action: setWindowProps({
                  label: this.win.label,
                  fullscreen: isFullscreen,
                }) as unknown as A,
                emitter: "WHITELIST",
              });
              if (this.fullscreenPoll != null) clearInterval(this.fullscreenPoll);
            }
          })
          .catch(console.error);
      }, 250);
    return await this.win.setFullscreen(value);
  }

  async center(): Promise<void> {
    return await this.win.center();
  }

  async setPosition(xy: xy.XY): Promise<void> {
    void this.win.setPosition(new LogicalPosition(xy.x, xy.y));
  }

  async setSize(dims: dimensions.Dimensions): Promise<void> {
    void this.win.setSize(new LogicalSize(dims.width, dims.height));
  }

  async setMinSize(dims: dimensions.Dimensions): Promise<void> {
    void this.win.setMinSize(new LogicalSize(dims.width, dims.height));
  }

  async setMaxSize(dims: dimensions.Dimensions): Promise<void> {
    void this.win.setMaxSize(new LogicalSize(dims.width, dims.height));
  }

  async setResizable(value: boolean): Promise<void> {
    // For some reason, listening to window resize events when the window is not
    // resizable causes issues. To resolve this, we unmount the listener
    if (TauriEventKey.WINDOW_RESIZED in this.unsubscribe && !value) {
      void this.unsubscribe[TauriEventKey.WINDOW_RESIZED]?.();
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
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

  async setDecorations(value: boolean): Promise<void> {
    return await this.win.setDecorations(value);
  }

  async getProps(): Promise<Omit<WindowProps, "key">> {
    const scaleFactor = await this.win.scaleFactor();
    const visible = await this.win.isVisible();
    return {
      position: await parsePosition(await this.win.innerPosition(), scaleFactor),
      size: await parseSize(await this.win.innerSize(), scaleFactor),
      maximized: await this.win.isMaximized(),
      visible,
      fullscreen: await this.win.isFullscreen(),
    };
  }
}

interface HandlerEntry {
  key: TauriEventKey;
  debounce: number;
  condition?: (win: WebviewWindow | null) => Promise<boolean>;
  handler: (ev: TauriEvent<any>) => Promise<AnyAction | null>;
}

const newWindowPropsHandlers = (): HandlerEntry[] => [
  {
    key: TauriEventKey.WINDOW_RESIZED,
    debounce: 200,
    handler: async (ev) => {
      const window = WebviewWindow.getByLabel(ev.windowLabel);
      if (window == null) return null;
      const scaleFactor = await window.scaleFactor();
      const visible = await window.isVisible();
      const nextProps: SetWindowPropsPayload = {
        label: ev.windowLabel,
        maximized: await window.isMaximized(),
        visible,
        minimized: !visible,
        position: await parsePosition(await window.innerPosition(), scaleFactor),
        size: await parseSize(await window.innerSize(), scaleFactor),
      };
      return setWindowProps(nextProps);
    },
  },
  {
    key: TauriEventKey.WINDOW_MOVED,
    debounce: 200,
    handler: async (ev) => {
      const window = WebviewWindow.getByLabel(ev.windowLabel);
      if (window == null) return null;
      const scaleFactor = await window?.scaleFactor();
      if (scaleFactor == null) return null;
      const position = await parsePosition(await window.innerPosition(), scaleFactor);
      const visible = await window.isVisible();
      const nextProps: SetWindowPropsPayload = {
        label: ev.windowLabel,
        visible,
        position,
      };
      return setWindowProps(nextProps);
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

const parsePosition = async (
  position: PhysicalPosition,
  scaleFactor: number
): Promise<xy.XY> => {
  const logical = position.toLogical(scaleFactor);
  return { x: logical.x, y: logical.y };
};

const parseSize = async (
  size: PhysicalSize,
  scaleFactor: number
): Promise<dimensions.Dimensions> => {
  const logical = size.toLogical(scaleFactor);
  return { width: logical.width, height: logical.height };
};
