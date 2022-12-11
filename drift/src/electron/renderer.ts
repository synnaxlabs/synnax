import { Action, AnyAction } from "@reduxjs/toolkit";
import { ipcRenderer } from "electron";

import { Event, Runtime } from "@/runtime";
import { actionEvent, driftKeyArgv } from "./util";
import { StoreState } from "@/state";
import { decode } from "@/serialization";

export default class ElectronRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  key(): string {
    return process.argv
      .find((arg) => arg.startsWith(driftKeyArgv))
      ?.slice(driftKeyArgv.length) as string;
  }

  isMain(): boolean {
    return false;
  }

  ready(): void {
    // do nothing
  }

  create(): void {
    throw new Error("Method not implemented.");
  }

  emit(event_: Omit<Event<S, A>, "emitter">, to?: string): void {
    ipcRenderer.send(actionEvent, { ...event_, emitter: this.key() }, to);
  }

  subscribe(lis: (event: Event<S, A>) => void): void {
    ipcRenderer.on(actionEvent, (event, event_: string) => lis(decode(event_)));
  }

  close(key: string): void {
    throw new Error("Method not implemented.");
  }

  focus(key: string): void {
    throw new Error("Method not implemented.");
  }

  onCloseRequested(cb: () => void): void {
    // do nothing
  }

  exists(key: string): boolean {
    throw new Error("Method not implemented.");
  }
}
