import type { Action, AnyAction } from "@reduxjs/toolkit";
import type { Event as TauriEvent, UnlistenFn } from "@tauri-apps/api/event";
import { listen, emit } from "@tauri-apps/api/event";
import { WebviewWindow, appWindow } from "@tauri-apps/api/window";

import { Event, Runtime } from "@/runtime";
import { decode, encode } from "@/serialization";
import { StoreState } from "@/state";
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
  private readonly window: WebviewWindow;
  private unsubscribe: UnlistenFn | undefined;

  /**
   * @param window - The WebviewWindow to use as the underlying engine for this runtime.
   * This should not be set in 99% of cases. Only use this if you know what you're doing.
   */
  constructor(window?: WebviewWindow) {
    this.window = window ?? appWindow;
  }

  key(): string {
    return this.window.label;
  }

  isMain(): boolean {
    return this.window.label === MAIN_WINDOW;
  }

  release(): void {
    this.unsubscribe?.();
  }

  ready(): void {
    void this.window.show();
  }

  create({ key, ...props }: KeyedWindowProps): void {
    const w = new WebviewWindow(key, {
      ...props,
      visible: false,
    });
    void w.once(tauriError, console.error);
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
    void listen<string>(actionEvent, (event: TauriEvent<string>) =>
      lis(decode(event.payload))
    )
      .catch(console.error)
      .then((unlisten) => {
        if (unlisten != null) this.unsubscribe = unlisten;
      });
  }

  onCloseRequested(cb: () => void): void {
    void this.window.onCloseRequested((e) => {
      // Only propagate the close request if the event
      // is for the current window.
      if (e.windowLabel === this.key()) {
        // Prevent default so the window doesn't close
        // until all processes are complete.
        e.preventDefault();
        cb();
      }
    });
  }

  close(key: string): void {
    const win = WebviewWindow.getByLabel(key);
    if (win != null) void win.close();
  }

  focus(key: string): void {
    const win = WebviewWindow.getByLabel(key);
    if (win != null) void win.setFocus();
  }

  exists(key: string): boolean {
    return !(WebviewWindow.getByLabel(key) == null);
  }
}
