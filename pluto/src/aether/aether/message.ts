// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors } from "@synnaxlabs/x";

import { type state } from "@/state";

/** Main → worker: create or update the component at `path` with `state`. */
export interface MainUpdateRequest {
  variant: "update";
  path: readonly string[];
  type: string;
  state: state.State;
}

/** Main → worker: delete the component at `path`. */
export interface MainDeleteRequest {
  variant: "delete";
  path: readonly string[];
}

/** Main → worker: invoke `method` on the component at `path` with `args`. `key`
 * correlates a response on {@link WorkerInvokeResponse}; omit for fire-and-forget. */
export interface MainInvokeRequest {
  variant: "invoke_request";
  key?: string;
  path: readonly string[];
  method: string;
  args: unknown[];
}

/** Worker → main: replace the state of the component at `path`. */
export interface WorkerUpdateRequest {
  variant: "update";
  path: readonly string[];
  state: state.State;
}

/** Worker → main: a worker-side error to surface on the main thread. */
export interface WorkerNotifyErrorRequest {
  variant: "error";
  error: errors.Payload;
}

/** Worker → main: response to a {@link MainInvokeRequest}. `error` is set when the
 * handler threw; otherwise `result` carries the return value. */
export interface WorkerInvokeResponse {
  variant: "invoke_response";
  key: string;
  result: unknown;
  error?: errors.Payload;
}

/** Any message sent from the worker thread to the main thread. */
export type WorkerMessage =
  | WorkerUpdateRequest
  | WorkerNotifyErrorRequest
  | WorkerInvokeResponse;

/** Any message sent from the main thread to the worker thread. */
export type MainMessage = MainUpdateRequest | MainDeleteRequest | MainInvokeRequest;

/** Send-only channel from the worker side. Held by individual aether components to post
 * {@link WorkerMessage}s back to the main thread. */
export interface Sender {
  send: (value: WorkerMessage, transfer?: Transferable[]) => void;
}

/** Bidirectional comms on the worker side. Sends {@link WorkerMessage}, handles
 * {@link MainMessage}. Consumed by {@link aether.render}. */
export interface WorkerComms extends Sender {
  handle: (handler: (value: MainMessage) => void) => void;
}

/** Bidirectional comms on the main side. Sends {@link MainMessage}, handles
 * {@link WorkerMessage}. Consumed by {@link Aether.Provider} and {@link Store}. */
export interface MainComms {
  send: (value: MainMessage, transfer?: Transferable[]) => void;
  handle: (handler: (value: WorkerMessage) => void) => void;
}

/** Sentinel used when `workerEnabled: false`. Lets the store stay non-null without
 * runtime null checks; any other missing-worker configuration is a constructor-time
 * error. */
export const NOOP_MAIN_COMMS: MainComms = { send: () => {}, handle: () => {} };

/** Adapts a `Worker` to {@link MainComms} for main-thread use. */
export const wrapWorker = (worker: Worker): MainComms => ({
  send: (value, transfer = []) => worker.postMessage(value, transfer),
  handle: (handler) => {
    worker.onmessage = (e: MessageEvent<WorkerMessage>) => handler(e.data);
  },
});

/** Returns {@link WorkerComms} bound to the dedicated worker global scope. The
 * worker-side parallel of {@link wrapWorker}; must be called from inside a dedicated
 * worker. */
export const wrapWorkerScope = (): WorkerComms => ({
  send: (value, transfer = []) => postMessage(value, { transfer }),
  handle: (handler) => {
    onmessage = (e: MessageEvent<MainMessage>) => handler(e.data);
  },
});

/** Creates a paired `[workerSide, mainSide]` of comms that route to each other. Use in
 * tests in place of a real {@link Worker}; the tuple order matches the direction
 * expected by {@link aether.render} and {@link Aether.Provider}. */
export const createMockPair = (): [WorkerComms, MainComms] => {
  let workerHandler: ((value: MainMessage) => void) | null = null;
  let mainHandler: ((value: WorkerMessage) => void) | null = null;
  return [
    {
      send: (value) => mainHandler?.(value),
      handle: (handler) => {
        workerHandler = handler;
      },
    },
    {
      send: (value) => workerHandler?.(value),
      handle: (handler) => {
        mainHandler = handler;
      },
    },
  ];
};
