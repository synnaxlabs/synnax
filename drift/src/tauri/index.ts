// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";
import { debounce as debounceF, type dimensions, type xy } from "@synnaxlabs/x";
import {
  emit,
  type Event as TauriEvent,
  listen,
  TauriEvent as TauriEventKey,
  type UnlistenFn,
} from "@tauri-apps/api/event";
import {
  getAllWebviewWindows as getAll,
  getCurrentWebviewWindow as getCurrentWindow,
  WebviewWindow,
} from "@tauri-apps/api/webviewWindow";
import {
  LogicalPosition,
  LogicalSize,
  type PhysicalPosition,
  type PhysicalSize,
} from "@tauri-apps/api/window";

import { type Event, type Runtime } from "@/runtime";
import { decode, encode } from "@/serialization";
import { setWindowProps, type SetWindowPropsPayload, type StoreState } from "@/state";
import { MAIN_WINDOW, type WindowProps } from "@/window";

const actionEvent = "drift://action";
const tauriError = "tauri://error";
const tauriCreated = "tauri://created";
const notFound = (key: string): Error => new Error(`Window not found: ${key}`);

//  Prevent the user or a programming error from creating a tiny window.
const MIN_DIM = 100;

const clampDims = (dims?: dimensions.Dimensions): dimensions.Dimensions | undefined => {
  if (dims == null) return undefined;
  return {
    width: Math.max(dims.width, MIN_DIM),
    height: Math.max(dims.height, MIN_DIM),
  };
};

const capWindowDimensions = (
  props: Omit<WindowProps, "key">,
): Omit<WindowProps, "key"> => {
  const { size, maxSize } = props;
  return { ...props, maxSize: clampDims(maxSize), size: clampDims(size) };
};

/**
 * A Tauri backed implementation of the drift Runtime.
 */
export class TauriRuntime<S extends StoreState, A extends Action = UnknownAction>
  implements Runtime<S, A>
{
  private readonly win: WebviewWindow;
  private unsubscribe: Record<string, UnlistenFn>;
  private fullscreenPoll: NodeJS.Timeout | null = null;

  /**
   * @param window - The WebviewWindow to use as the underlying engine for this runtime.
   * This should not be set in 99% of cases. Only use this if you know what you're doing.
   */
  constructor(window?: WebviewWindow) {
    this.win = window ?? getCurrentWindow();
    this.unsubscribe = {};
  }

  async configure(): Promise<void> {
    let prevFullscreen = (await this.getProps()).fullscreen;
    this.fullscreenPoll = setInterval(() => {
      this.win
        .isFullscreen()
        .then((isFullscreen) => {
          if (isFullscreen !== prevFullscreen) {
            prevFullscreen = isFullscreen;
            this.emit(
              {
                action: setWindowProps({
                  label: this.win.label,
                  fullscreen: isFullscreen,
                }) as unknown as A,
              },
              undefined,
              "WHITELIST",
            );
          }
        })
        .catch(console.error);
    }, 250);
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

  async emit(
    event_: Omit<Event<S, A>, "emitter">,
    to?: string,
    emitter: string = this.label(),
  ): Promise<void> {
    const event = encode({ ...event_, emitter });
    if (to == null) return await emit(actionEvent, event);
    const win = await WebviewWindow.getByLabel(to);
    if (win == null) throw notFound(to);
    await win.emit(actionEvent, event);
  }

  async subscribe(lis: (action: Event<S, A>) => void): Promise<void> {
    this.release();
    this.unsubscribe[actionEvent] = await listen<string>(
      actionEvent,
      (event: TauriEvent<string>) => lis(decode(event.payload)),
    );
    const propsHandlers = newWindowPropsHandlers();
    for (const { key, handler, debounce } of propsHandlers) 
      this.unsubscribe[key] = await this.win.listen(
        key,
        debounceF(() => {
          handler(this.win)
            .then((action) => {
              if (action != null) 
                this.emit({ action: action as A }, undefined, "WHITELIST");
              
            })
            .catch(console.error);
        }, debounce),
      );
    
  }

  onCloseRequested(cb: () => void): void {
    void this.win.onCloseRequested((e) => {
      e.preventDefault();
      cb();
    });
  }

  async create(label: string, props: Omit<WindowProps, "key">): Promise<void> {
    const { size, minSize, maxSize, position, ...rest } = capWindowDimensions(props);
    if (size?.width != null) size.width = Math.max(size.width, MIN_DIM);
    if (size?.height != null) size.height = Math.max(size.height, MIN_DIM);
    if (maxSize?.width != null) maxSize.width = Math.max(maxSize.width, MIN_DIM);
    if (maxSize?.height != null) maxSize.height = Math.max(maxSize.height, MIN_DIM);
    try {
      const w = new WebviewWindow(label, {
        x: position?.x,
        y: position?.y,
        width: size?.width,
        height: size?.height,
        minWidth: minSize?.width,
        minHeight: minSize?.height,
        maxWidth: maxSize?.width,
        maxHeight: maxSize?.height,
        titleBarStyle: "overlay",
        dragDropEnabled: false,
        ...rest,
      });
      return await new Promise<void>((resolve, reject) => {
        void w.once(tauriError, (e) => reject(e.payload));
        void w.once(tauriCreated, () => resolve());
      });
    } catch (e) {
      console.error(e);
    }
  }

  async close(label: string): Promise<void> {
    const win = await WebviewWindow.getByLabel(label);
    if (win != null)
      try {
        await win.destroy();
      } catch (e) {
        console.error(e, label);
      }
    else {
      const wins = await getAll();
      console.error(
        "Window not found",
        label,
        wins.map((w) => w.label),
      );
    }
  }

  async listLabels(): Promise<string[]> {
    const res = await getAll();
    return res.map((w) => w.label);
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
    await this.win.setFullscreen(value);
  }

  async center(): Promise<void> {
    return await this.win.center();
  }

  async setPosition(xy: xy.XY): Promise<void> {
    await this.win.setPosition(new LogicalPosition(xy.x, xy.y));
  }

  async setSize(dims: dimensions.Dimensions): Promise<void> {
    dims = clampDims(dims) as dimensions.Dimensions;
    await this.win.setSize(new LogicalSize(dims.width, dims.height));
  }

  async setMinSize(dims: dimensions.Dimensions): Promise<void> {
    dims = clampDims(dims) as dimensions.Dimensions;
    await this.win.setMinSize(new LogicalSize(dims.width, dims.height));
  }

  async setMaxSize(dims: dimensions.Dimensions): Promise<void> {
    dims = clampDims(dims) as dimensions.Dimensions;
    await this.win.setMaxSize(new LogicalSize(dims.width, dims.height));
  }

  async setResizable(value: boolean): Promise<void> {
    // For some reason, listening to window resize events when the window is not
    // resizable causes issues. To resolve this, we unmount the listener
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
  handler: (win: WebviewWindow) => Promise<UnknownAction | null>;
}

const newWindowPropsHandlers = (): HandlerEntry[] => [
  {
    key: TauriEventKey.WINDOW_RESIZED,
    debounce: 200,
    handler: async (window) => {
      const scaleFactor = await window.scaleFactor();
      const visible = await window.isVisible();
      const nextProps: SetWindowPropsPayload = {
        label: window.label,
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
    handler: async (window) => {
      const scaleFactor = await window?.scaleFactor();
      if (scaleFactor == null) return null;
      const position = await parsePosition(await window.innerPosition(), scaleFactor);
      const visible = await window.isVisible();
      const nextProps: SetWindowPropsPayload = {
        label: window.label,
        visible,
        position,
      };
      return setWindowProps(nextProps);
    },
  },
  {
    key: TauriEventKey.WINDOW_BLUR,
    debounce: 0,
    handler: async (window) => setWindowProps({ focus: false, label: window.label }),
  },
  {
    key: TauriEventKey.WINDOW_FOCUS,
    debounce: 0,
    handler: async (window) => setWindowProps({
        focus: true,
        visible: true,
        minimized: false,
        label: window.label,
      }),
  },
];

const parsePosition = async (
  position: PhysicalPosition,
  scaleFactor: number,
): Promise<xy.XY> => {
  const logical = position.toLogical(scaleFactor);
  return { x: logical.x, y: logical.y };
};

const parseSize = async (
  size: PhysicalSize,
  scaleFactor: number,
): Promise<dimensions.Dimensions> => {
  const logical = size.toLogical(scaleFactor);
  return { width: logical.width, height: logical.height };
};
