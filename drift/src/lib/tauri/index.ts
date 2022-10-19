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
  Runtime as DriftRuntime,
} from '../runtime';

const actionEvent = 'drift:action';
const tauriWindowCreated = 'tauri://created';
const tauriError = 'tauri://error';

export class Runtime implements DriftRuntime {
  unsubscribe?: void | UnlistenFn;

  ready(): void {
    appWindow.show();
  }

  isMain(): boolean {
    return appWindow.label === 'main';
  }

  release() {
    this.unsubscribe && this.unsubscribe();
  }

  createWindow({ key, ...props }: KeyedWindowProps) {
    return new Promise<Window>((resolve, reject) => {
      const w = new WebviewWindow(key as string, {
        ...props,
        visible: false,
      });
      w.once(tauriWindowCreated, () => {
        resolve(w as unknown as Window);
      });
      w.once(tauriError, (err: unknown) => {
        reject(err);
      });
    });
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

  winKey(): string {
    return appWindow.label;
  }
}
