import { Action, AnyAction } from '@reduxjs/toolkit';
import {
  emit,
  listen,
  Event as TauriEvent,
  UnlistenFn,
} from '@tauri-apps/api/event';
import { appWindow, WebviewWindow } from '@tauri-apps/api/window';

import { StoreState } from '../slice';
import {
  Event as DriftEvent,
  Window as DriftWindow,
  KeyedWindowProps,
} from '../window';

const actionEvent = 'action';
const tauriError = 'tauri://error';

export default class Window<S extends StoreState, A extends Action = AnyAction>
  implements DriftWindow<S, A>
{
  window: WebviewWindow;
  unsubscribe?: void | UnlistenFn;
  nextClose: boolean;

  constructor(window?: WebviewWindow) {
    this.window = window || appWindow;
    this.nextClose = false;
  }

  key(): string {
    return this.window.label;
  }

  isMain(): boolean {
    return this.window.label === 'main';
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

  emit(event: DriftEvent<S, A>): void {
    emit(actionEvent, event);
  }

  subscribe(lis: (action: DriftEvent<S, A>) => void): void {
    listen<string>(actionEvent, (event: TauriEvent<string>) => {
      lis(JSON.parse(event.payload) as DriftEvent<S, A>);
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
