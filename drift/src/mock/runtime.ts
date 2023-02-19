// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, AnyAction } from "@reduxjs/toolkit";

import { Event, Runtime } from "@/runtime";
import { StoreState } from "@/state";
import { KeyedWindowProps } from "@/window";

export class MockRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  _isMain = false;
  _key = "mock";
  markedReady = false;
  emissions: Array<Event<S, A>> = [];
  hasCreated: KeyedWindowProps[] = [];
  hasClosed: string[] = [];
  hasFocused: string[] = [];
  subscribeCallback: (event: Event<S, A>) => void = () => {};
  requestClosure: () => void = () => {};

  constructor(isMain: boolean) {
    this._isMain = isMain;
  }

  isMain(): boolean {
    return this._isMain;
  }

  key(): string {
    return this._key;
  }

  emit(event: Omit<Event<S, A>, "emitter">, to?: string): void {
    this.emissions.push({ ...event, emitter: this.key() });
  }

  subscribe(lis: (event: Event<S, A>) => void): void {
    this.subscribeCallback = lis;
  }

  ready(): void {
    this.markedReady = true;
  }

  create(props: KeyedWindowProps): void {
    this.hasCreated.push(props);
  }

  close(key: string): void {
    this.hasClosed.push(key);
  }

  onCloseRequested(cb: () => void): void {
    this.requestClosure = cb;
  }

  focus(key: string): void {
    this.hasFocused.push(key);
  }

  exists(key: string): boolean {
    // check if in list of created and NOT in list of closed
    const hasBeenCreated = this.hasCreated.some((w) => w.key === key);
    const hasBeenClosed = this.hasClosed.some((w) => w === key);
    return hasBeenCreated && !hasBeenClosed;
  }
}
