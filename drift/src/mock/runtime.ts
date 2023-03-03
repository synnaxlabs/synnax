// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, AnyAction } from "@reduxjs/toolkit";
import { Dimensions, XY } from "@synnaxlabs/x";

import { Event, Runtime } from "@/runtime";
import { StoreState } from "@/state";
import { KeyedWindowProps, WindowProps } from "@/window";

export class MockRuntime<S extends StoreState, A extends Action = AnyAction>
  implements Runtime<S, A>
{
  _isMain = false;
  _key = "mock";
  markedReady = false;
  hasClosed: string[] = [];
  emissions: Array<Event<S, A>> = [];
  hasCreated: KeyedWindowProps[] = [];
  subscribeCallback: (event: Event<S, A>) => void = () => {};
  requestClosure: () => void = () => {};
  props: WindowProps;

  constructor(isMain: boolean) {
    this._isMain = isMain;
    this.props = {};
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

  onCloseRequested(cb: () => void): void {
    this.requestClosure = cb;
  }

  // |||||| MANAGER IMPLEMENTATION ||||||

  create(props: KeyedWindowProps): void {
    this.hasCreated.push(props);
  }

  async close(key: string): Promise<void> {
    this.hasClosed.push(key);
  }

  async focus(): Promise<void> {
    this.props.focus = true;
    return await Promise.resolve();
  }

  async setMinimized(value: boolean): Promise<void> {
    this.props.visible = !value;
    return await Promise.resolve();
  }

  async setMaximized(value: boolean): Promise<void> {
    this.props.maximized = value;
    return await Promise.resolve();
  }

  async setVisible(value: boolean): Promise<void> {
    this.props.visible = value;
    return await Promise.resolve();
  }

  async setFullscreen(value?: boolean): Promise<void> {
    this.props.fullscreen = value ?? !(this.props.fullscreen ?? false);
    return await Promise.resolve();
  }

  async center(): Promise<void> {
    this.props.center = true;
    return await Promise.resolve();
  }

  async setPosition(xy: XY): Promise<void> {
    this.props.x = xy.x;
    this.props.y = xy.y;
    return await Promise.resolve();
  }

  async setSize(dims: Dimensions): Promise<void> {
    this.props.width = dims.width;
    this.props.height = dims.height;
    return await Promise.resolve();
  }

  async setMinSize(dims: Dimensions): Promise<void> {
    this.props.minWidth = dims.width;
    this.props.minHeight = dims.height;
    return await Promise.resolve();
  }

  async setMaxSize(dims: Dimensions): Promise<void> {
    this.props.maxWidth = dims.width;
    this.props.maxHeight = dims.height;
    return await Promise.resolve();
  }

  async setResizable(value?: boolean): Promise<void> {
    this.props.resizable = value ?? !(this.props.resizable ?? false);
    return await Promise.resolve();
  }

  async setSkipTaskbar(value?: boolean): Promise<void> {
    this.props.skipTaskbar = value ?? !(this.props.skipTaskbar ?? false);
    return await Promise.resolve();
  }

  async setAlwaysOnTop(_value?: boolean): Promise<void> {
    return await Promise.resolve();
  }

  async setTitle(title: string): Promise<void> {
    this.props.title = title;
    return await Promise.resolve();
  }

  async show(): Promise<void> {
    this.props.visible = true;
    return await Promise.resolve();
  }
}
