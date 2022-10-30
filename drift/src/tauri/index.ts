import { Action, AnyAction } from '@reduxjs/toolkit';
import {
  emit,
  listen,
  Event as TauriEvent,
  UnlistenFn,
} from '@tauri-apps/api/event';
import { appWindow, WebviewWindow } from '@tauri-apps/api/window';

import { Event, Runtime } from '../runtime';
import { StoreState } from '../state';
import { KeyedWindowProps, MAIN_WINDOW } from '../window';

const actionEvent = 'action';
const tauriError = 'tauri://error';
const notFound = (key: string) => new Error(`Window not found: ${key}`);

/**
 * A Tauri backed implementation of the drift Runtime.
 */
export class TauriRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  private window: WebviewWindow;
  private unsubscribe?: void | UnlistenFn;

  /**
   * @param window - The WebviewWindow to use as the underlying engine for this runtime.
   * This should not be set in 99% of cases. Only use this if you know what you're doing.
   */
  constructor(window?: WebviewWindow) {
    this.window = window || appWindow;
  }

  key(): string {
    return this.window.label;
  }

  isMain(): boolean {
    return this.window.label === MAIN_WINDOW;
  }

  release() {
    this.unsubscribe && this.unsubscribe();
  }

  ready(): void {
    this.window.show();
  }

  createWindow({ key, ...props }: KeyedWindowProps) {
    const w = new WebviewWindow(key as string, {
      ...props,
      visible: false,
    });
    w.once(tauriError, console.error);
  }

  emit(event: Omit<Event<S, A>, 'emitter'>, to?: string): void {
    let e = emit;
    if (to) {
      const win = WebviewWindow.getByLabel(to);
      if (!win) throw notFound(to);
      e = win.emit;
    }
    e(actionEvent, { ...event, emitter: this.key() });
  }

  subscribe(lis: (action: Event<S, A>) => void): void {
    listen<string>(actionEvent, (event: TauriEvent<string>) => {
      lis(JSON.parse(event.payload) as Event<S, A>);
    })
      .catch(console.error)
      .then((unlisten) => {
        this.unsubscribe = unlisten;
      });
  }

  onCloseRequested(cb: () => void): void {
    this.window.onCloseRequested((e) => {
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
    if (win) win.close();
  }

  focus(key: string): void {
    const win = WebviewWindow.getByLabel(key);
    if (win) win.setFocus();
  }

  exists(key: string): boolean {
    return !!WebviewWindow.getByLabel(key);
  }
}
