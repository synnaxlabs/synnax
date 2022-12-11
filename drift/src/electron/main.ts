import { Action, AnyAction } from "@reduxjs/toolkit";
import { BrowserWindow, ipcMain, app } from "electron";

import { Event, Runtime } from "@/runtime";
import { StoreState } from "@/state";
import { KeyedWindowProps, MAIN_WINDOW } from "@/window";
import { encode } from "@/serialization";

import { actionEvent, driftKeyArgv } from "./util";
import { i } from "vitest/dist/index-40e0cb97";

export default class ElectronRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  windows: Map<string, BrowserWindow> = new Map();
  lis: (action: Event<S, A>) => void = () => {};

  constructor() {
    ipcMain.on(actionEvent, (event, event_: Event<S, A>, target?: string) => {
      this.emitInternal(event_, target);
      this.lis(event_);
    });
  }

  key(): string {
    return MAIN_WINDOW;
  }

  isMain(): boolean {
    return true;
  }

  ready(): void {
    // do nothing
  }

  create({ key, ...props }: KeyedWindowProps) {
    const { url, focus, decorations, fileDropEnabled, visible } = props;
    const bw = new BrowserWindow({
      webPreferences: {
        additionalArguments: [`${driftKeyArgv}${key}`],
      },
      show: visible,
      ...props,
    });
    bw.loadURL(url as string);
    this.windows.set(key, bw);
  }

  emit(event_: Omit<Event<S, A>, "emitter">, to?: string): void {
    this.emitInternal({ ...event_, emitter: this.key() }, to);
  }

  subscribe(lis: (event: Event<S, A>) => void): void {
    this.lis = lis;
  }

  onCloseRequested(cb: () => void): void {
    // do nothing
  }

  private emitInternal(event_: Event<S, A>, to?: string): void {
    let windows = [...this.windows.values()];
    if (to) {
      const win = this.windows.get(to);
      if (!win) throw new Error(`Window ${to} not found`);
      windows = [win];
    }
    windows.forEach((w) => w.webContents.send(actionEvent, encode(event_)));
  }

  close(key: string): void {
    if (key == this.key()) app.quit();
    else this.windows.get(key)?.close();
  }

  focus(key: string): void {
    this.windows.get(key)?.focus();
  }

  exists(key: string): boolean {
    return this.windows.has(key);
  }
}
