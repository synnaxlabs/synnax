// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";
import {
  box,
  debounce as debounceF,
  deep,
  dimensions,
  runtime,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
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
  availableMonitors,
  LogicalPosition,
  LogicalSize,
  type PhysicalPosition,
  type PhysicalSize,
} from "@tauri-apps/api/window";

import { type Event, type Runtime } from "@/runtime";
import { decode, encode } from "@/serialization";
import {
  runtimeSetWindowProps,
  type RuntimeSetWindowProsPayload,
  type StoreState,
} from "@/state";
import { MAIN_WINDOW, type WindowProps } from "@/window";

const actionEvent = "drift://action";
const tauriError = "tauri://error";
const tauriCreated = "tauri://created";
const notFound = (key: string): Error => new Error(`Window not found: ${key}`);

//  Prevent the user or a programming error from creating a tiny window.
const MIN_DIM = 250;

// On macOS, we need to poll for fullscreen changes, as tauri doesn't provide an event
// for it. This is the interval at which we poll.
const MACOS_FULLSCREEN_POLL_INTERVAL = TimeSpan.seconds(1);

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

/** @returns the bounding boxes for all available monitors. */
const monitorBoxes = async (): Promise<box.Box[]> => {
  const monitors = await availableMonitors();
  return monitors.map((monitor) => {
    const pos = parsePosition(monitor.position, monitor.scaleFactor);
    const dims = { width: monitor.size.width, height: monitor.size.height };
    return box.construct(pos, dims);
  });
};

/**
 * @returns true whether the top-left corner of the window is visible on the user's
 * monitors.
 */
const isPositionVisible = async (position: xy.XY): Promise<boolean> => {
  const boxes = await monitorBoxes();
  return boxes.some((b) => box.contains(b, position));
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
    // We only need to poll for fullscreen on macOS, as tauri doesn't provide an
    // emitted event for fullscreen changes.
    await this.startFullscreenPoll();
  }

  private async startFullscreenPoll(): Promise<void> {
    if (runtime.getOS() !== "macOS") return;
    let prevFullscreen = (await this.getProps()).fullscreen;
    if (this.fullscreenPoll != null) clearInterval(this.fullscreenPoll);
    this.fullscreenPoll = setInterval(() => {
      this.win
        .isFullscreen()
        .then((isFullscreen) => {
          if (isFullscreen !== prevFullscreen) {
            prevFullscreen = isFullscreen;
            this.emit(
              {
                action: runtimeSetWindowProps({
                  label: this.win.label,
                  fullscreen: isFullscreen,
                }) as unknown as A,
              },
              undefined,
              "WHITELIST",
            ).catch(console.error);
          }
        })
        .catch(console.error);
    }, MACOS_FULLSCREEN_POLL_INTERVAL.milliseconds);
  }

  label(): string {
    return this.win.label;
  }

  isMain(): boolean {
    return this.win.label === MAIN_WINDOW;
  }

  release(): void {
    Object.values(this.unsubscribe).forEach((f) => f?.());
    if (this.fullscreenPoll != null) clearInterval(this.fullscreenPoll);

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
    await this.startFullscreenPoll();
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
                this.emit({ action: action as A }, undefined, "WHITELIST").catch(
                  console.error,
                );
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
    props = deep.copy(props);
    const { size, minSize, maxSize, position, ...rest } = capWindowDimensions(props);
    if (size?.width != null) size.width = Math.max(size.width, MIN_DIM);
    if (size?.height != null) size.height = Math.max(size.height, MIN_DIM);
    if (maxSize?.width != null) maxSize.width = Math.max(maxSize.width, MIN_DIM);
    if (maxSize?.height != null) maxSize.height = Math.max(maxSize.height, MIN_DIM);
    if (position != null) {
      const isVisible = await isPositionVisible(position);
      if (!isVisible) {
        position.x = 0;
        position.y = 0;
      }
    }
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
        void w.once(tauriError, (e) => reject(new Error(JSON.stringify(e.payload))));
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
    const logicalPos = new LogicalPosition(xy.x, xy.y);
    const isVisible = await isPositionVisible(xy);
    if (!isVisible) {
      logicalPos.x = 0;
      logicalPos.y = 0;
    }
    await this.win.setPosition(logicalPos);
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
      position: parsePosition(await this.win.innerPosition(), scaleFactor),
      size: parseSize(await this.win.innerSize(), scaleFactor),
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
      const nextProps: RuntimeSetWindowProsPayload = {
        label: window.label,
        maximized: await window.isMaximized(),
        visible,
        minimized: !visible,
        position: parsePosition(await window.innerPosition(), scaleFactor),
        size: parseSize(await window.innerSize(), scaleFactor),
      };
      return runtimeSetWindowProps(nextProps);
    },
  },
  {
    key: TauriEventKey.WINDOW_MOVED,
    debounce: 200,
    handler: async (window) => {
      const scaleFactor = await window?.scaleFactor();
      if (scaleFactor == null) return null;
      const position = parsePosition(await window.innerPosition(), scaleFactor);
      const visible = await window.isVisible();
      const nextProps: RuntimeSetWindowProsPayload = {
        label: window.label,
        visible,
        position,
      };
      return runtimeSetWindowProps(nextProps);
    },
  },
  {
    key: TauriEventKey.WINDOW_BLUR,
    debounce: 0,
    handler: async (window) =>
      runtimeSetWindowProps({ focus: false, label: window.label }),
  },
  {
    key: TauriEventKey.WINDOW_FOCUS,
    debounce: 0,
    handler: async (window) =>
      runtimeSetWindowProps({
        focus: true,
        visible: true,
        minimized: false,
        label: window.label,
      }),
  },
];

const parsePosition = (position: PhysicalPosition, scaleFactor: number): xy.XY =>
  xy.scale(position, 1 / scaleFactor);

const parseSize = (size: PhysicalSize, scaleFactor: number): dimensions.Dimensions =>
  dimensions.scale(size, 1 / scaleFactor);
