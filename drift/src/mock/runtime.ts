// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, UnknownAction } from "@reduxjs/toolkit";
import { type dimensions, type xy } from "@synnaxlabs/x";

import { type Event, type Runtime } from "@/runtime";
import { type StoreState } from "@/state";
import { type WindowProps } from "@/window";

export class MockRuntime<S extends StoreState, A extends Action = UnknownAction>
  implements Runtime<S, A>
{
  _isMain = false;
  _label = "mock";
  markedReady = false;
  hasClosed: string[] = [];
  emissions: Array<Event<S, A>> = [];
  hasCreated: Record<string, Omit<WindowProps, "key">> = {};
  subscribeCallback: (event: Event<S, A>) => void = () => {};
  requestClosure: () => void = () => {};
  props: WindowProps;

  constructor(isMain: boolean, initialProps: WindowProps = { key: "mock" }) {
    this._isMain = isMain;
    this.props = { ...initialProps };
    this._label = initialProps.key;
  }

  async configure(): Promise<void> {}

  isMain(): boolean {
    return this._isMain;
  }

  label(): string {
    return this._label;
  }

  async emit(event: Omit<Event<S, A>, "emitter">): Promise<void> {
    this.emissions.push({ ...event, emitter: this.label() });
    return await Promise.resolve();
  }

  async subscribe(lis: (event: Event<S, A>) => void): Promise<void> {
    this.subscribeCallback = lis;
  }

  onCloseRequested(cb: () => void): void {
    this.requestClosure = cb;
  }

  async listLabels(): Promise<string[]> {
    return [];
  }

  async create(label: string, props: Omit<WindowProps, "key">): Promise<void> {
    this.hasCreated[label] = props;
    return await Promise.resolve();
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

  async setFullscreen(value: boolean): Promise<void> {
    this.props.fullscreen = value;
    return await Promise.resolve();
  }

  async center(): Promise<void> {
    this.props.center = true;
    return await Promise.resolve();
  }

  async setPosition(xy: xy.XY): Promise<void> {
    this.props.position = xy;
    return await Promise.resolve();
  }

  async setSize(dims: dimensions.Dimensions): Promise<void> {
    this.props.size = dims;
    return await Promise.resolve();
  }

  async setMinSize(dims: dimensions.Dimensions): Promise<void> {
    this.props.minSize = dims;
    return await Promise.resolve();
  }

  async setMaxSize(dims: dimensions.Dimensions): Promise<void> {
    this.props.maxSize = dims;
    return await Promise.resolve();
  }

  async setResizable(value: boolean): Promise<void> {
    this.props.resizable = value;
    return await Promise.resolve();
  }

  async setSkipTaskbar(value: boolean): Promise<void> {
    this.props.skipTaskbar = value;
    return await Promise.resolve();
  }

  async setAlwaysOnTop(value: boolean): Promise<void> {
    this.props.alwaysOnTop = value;
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

  async setDecorations(value: boolean): Promise<void> {
    this.props.decorations = value;
    return await Promise.resolve();
  }

  async getProps(): Promise<Omit<WindowProps, "key">> {
    return this.props;
  }
}
