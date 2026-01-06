// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";
import { type dimensions, type xy } from "@synnaxlabs/x";

import { type StoreState } from "@/state";
import { type WindowProps } from "@/window";

/**
 * An event emitted by drift to communicate state changes.
 */
export interface Event<S extends StoreState, A extends Action = UnknownAction> {
  /** The key of the window that emitted the event */
  emitter: string;
  /** A redux state action */
  action?: A;
  /** The entire redux store state. Sent only on the creation of new windows */
  state?: S;
  /** sendState is set to true when the window is requesting a state forward */
  sendState?: boolean;
}

export interface Sender<S extends StoreState, A extends Action = UnknownAction> {
  /**
   * Emits an event to all windows in the application.
   * @param event - The event to emit.
   * @param to - If set, the event will only be emitted to the window with the given key.
   */
  emit: (event: Omit<Event<S, A>, "emitter">, to?: string) => Promise<void>;
}

export interface Receiver<S extends StoreState, A extends Action = UnknownAction> {
  /**
   * Listens for an event from any window in the application.
   * @param lis - The callback to call when the event is received.
   */
  subscribe: (lis: (event: Event<S, A>) => void) => Promise<void>;
}

export interface MainChecker {
  /**
   * @returns true if the window is the main window of the application i.e. the first
   * forked
   */
  isMain: () => boolean;
}

/**
 * Communicator allows for event communication between windows.
 */
export interface Communicator<S extends StoreState, A extends Action = UnknownAction>
  extends Sender<S, A>, Receiver<S, A>, MainChecker {}

/**
 * Properties represents the runtime properties of a window.
 */
export interface Properties {
  /**
   * @returns the key of the window.
   */
  label: () => string;
  /**
   * Calls the provided function with the current window is closing.
   */
  onCloseRequested: (cb: () => void) => void;
  listLabels: () => Promise<string[]>;
  getProps: () => Promise<Omit<WindowProps, "key">>;
}

/**
 * Manager is used to manage the windows in the application.
 */
export interface Manager {
  /**
   * Creates a new window with the given properties. The window should not be shown
   * until the ready() method is called.
   */
  create: (label: string, props: Omit<WindowProps, "key">) => Promise<void>;
  /**
   * Closes the window with the given key.
   */
  close: (label: string) => Promise<void>;
  /**
   * Focuses the window with the given key.
   */
  focus: () => Promise<void>;
  setMinimized: (value: boolean) => Promise<void>;
  setMaximized: (value: boolean) => Promise<void>;
  setVisible: (value: boolean) => Promise<void>;
  setFullscreen: (value: boolean) => Promise<void>;
  center: () => Promise<void>;
  setPosition: (xy: xy.XY) => Promise<void>;
  setSize: (dims: dimensions.Dimensions) => Promise<void>;
  setMinSize: (dims: dimensions.Dimensions) => Promise<void>;
  setMaxSize: (dims: dimensions.Dimensions) => Promise<void>;
  setResizable: (value: boolean) => Promise<void>;
  setSkipTaskbar: (value: boolean) => Promise<void>;
  setAlwaysOnTop: (value: boolean) => Promise<void>;
  setDecorations: (value: boolean) => Promise<void>;
  setTitle: (title: string) => Promise<void>;
  configure(): Promise<void>;
}

/**
 * An interface that represents the core runtime of the application.
 * Drift uses this runtime to manage windows and communicate between them.
 */
export interface Runtime<S extends StoreState, A extends Action = UnknownAction>
  extends Communicator<S, A>, Properties, Manager {}
