import { Action, AnyAction } from '@reduxjs/toolkit';

import { Event as DriftEvent, Runtime as DriftWindow } from '../runtime';
import { StoreState } from '../state';
import { KeyedWindowProps } from '../window';

export class WindowNetwork<S extends StoreState, A extends Action = AnyAction> {
  windows: { [key: string]: Window<S, A> } = {};

  create(key: string): Window<S, A> {
    const w = new Window<S, A>(this, key);
    this.windows[w.key()] = w;
    return w;
  }

  notify(event: DriftEvent<S, A>): void {
    Object.values(this.windows).forEach(
      (w) => w._listener && w._listener(event)
    );
  }

  close(key: string): void {
    delete this.windows[key];
  }

  exists(key: string): boolean {
    return !!this.windows[key];
  }
}

export default class Window<S extends StoreState, A extends Action = AnyAction>
  implements DriftWindow<S, A>
{
  private network: WindowNetwork<S, A>;
  private _key: string | null;
  _listener: ((action: DriftEvent<S, A>) => void) | null;
  _closeCb: (() => void) | null = null;

  constructor(network: WindowNetwork<S, A>, key: string) {
    this.network = network;
    this._key = key;
    this._listener = null;
    this._closeCb = null;
  }

  isMain(): boolean {
    return this._key === 'main';
  }

  release(): void {
    // do nothing
  }

  ready(): void {
    // do nothing
  }

  key(): string {
    return this._key || '';
  }

  createWindow({ key }: KeyedWindowProps) {
    this.network.create(key);
  }

  emit(event: DriftEvent<S, A>): void {
    this.network.notify(event);
  }

  subscribe(lis: (action: DriftEvent<S, A>) => void): void {
    this._listener = lis;
  }

  onCloseRequested(cb: () => void): void {
    this._closeCb = cb;
  }

  close(key: string): void {
    this.network.close(key);
  }

  focus(): void {
    // do nothing
  }

  exists(key: string): boolean {
    return this.network.exists(key);
  }
}
