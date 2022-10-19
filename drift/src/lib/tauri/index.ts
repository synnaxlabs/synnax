import {
  listen,
  emit,
  Event as TauriEvent,
  UnlistenFn,
} from '@tauri-apps/api/event';
import { appWindow, WebviewWindow } from '@tauri-apps/api/window';
import {
  KeyedWindowProps,
  Event as DriftEvent,
  Window as DriftWindow,
} from '../window';

const actionEvent = 'drift:action';
const tauriError = 'tauri://error';

export default class Window implements DriftWindow {
  window: WebviewWindow;
  unsubscribe?: void | UnlistenFn;

  constructor(window?: WebviewWindow) {
    this.window = window || appWindow;
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

  emit(event: DriftEvent): void {
    emit(actionEvent, event);
  }

  subscribe(lis: (action: DriftEvent) => void): void {
    listen<string>(actionEvent, (event: TauriEvent<string>) => {
      lis(JSON.parse(event.payload) as DriftEvent);
    })
      .catch(console.error)
      .then((unlisten) => {
        this.unsubscribe = unlisten;
      });
  }

  onClose(cb: () => void): void {
    this.window.onCloseRequested(cb);
  }

  close(key: string): void {
    WebviewWindow.getByLabel(key)?.close();
  }
}
