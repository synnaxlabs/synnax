import { type errors, type state } from "@synnaxlabs/x";
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
/** Main → worker: delete every component in the tree, leaving the root usable. Sent on
 * store disposal, which for in-process comms is the only thing that tears the tree
 * down. */
export interface MainClearRequest {
    variant: "clear";
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
export type WorkerMessage = WorkerUpdateRequest | WorkerNotifyErrorRequest | WorkerInvokeResponse;
/** Any message sent from the main thread to the worker thread. */
export type MainMessage = MainUpdateRequest | MainDeleteRequest | MainClearRequest | MainInvokeRequest;
/** Send-only channel handed to individual aether components. Takes one
 * {@link WorkerMessage} at a time; the implementation decides when it reaches the wire. */
export interface Sender {
    send: (value: WorkerMessage, transfer?: Transferable[]) => void;
}
/** Bidirectional comms on the worker side. Consumed by {@link aether.render}. */
export interface WorkerComms {
    send: (values: WorkerMessage[], transfer?: Transferable[]) => void;
    handle: (handler: (values: MainMessage[]) => void) => void;
}
/** Bidirectional comms on the main side. Consumed by {@link Aether.Provider} and
 * {@link Store}. */
export interface MainComms {
    send: (values: MainMessage[], transfer?: Transferable[]) => void;
    handle: (handler: (values: WorkerMessage[]) => void) => void;
}
/** Sentinel used when `workerEnabled: false`. Lets the store stay non-null without
 * runtime null checks; any other missing-worker configuration is a constructor-time
 * error. */
export declare const NOOP_MAIN_COMMS: MainComms;
/** Adapts a `Worker` to {@link MainComms} for main-thread use. */
export declare const wrapWorker: (worker: Worker) => MainComms;
/** Returns {@link WorkerComms} bound to the dedicated worker global scope. The
 * worker-side parallel of {@link wrapWorker}; must be called from inside a dedicated
 * worker. */
export declare const wrapWorkerScope: () => WorkerComms;
/** Buffers messages and hands them to `flushTo` as one array on the next microtask.
 *
 * Both directions batch. `postMessage` has a fixed per-call cost that dominates the
 * payload at these sizes, and each delivery is one task on the receiving thread, so a
 * batch is also one React render instead of one per message. */
export declare class Batcher<M> {
    private readonly flushTo;
    private readonly beforeFlush;
    private buffer;
    private transfer;
    private scheduled;
    /** `beforeFlush` runs at the top of every flush; anything it sends joins the batch it
     * precedes. Use it for work that must lead the batch, such as ordering pending
     * creates. */
    constructor(flushTo: (values: M[], transfer: Transferable[]) => void, beforeFlush?: () => void);
    /** Buffers `value` for the next flush. Nothing is delivered synchronously. */
    send(value: M, transfer?: Transferable[]): void;
    /** Requests a flush even with an empty buffer, so `beforeFlush` gets to run.
     * Idempotent within a microtask. */
    schedule(): void;
    /** Drops everything buffered. */
    clear(): void;
    private flush;
}
/** Creates a paired `[workerSide, mainSide]` of comms that route to each other. Use in
 * tests in place of a real {@link Worker}; the tuple order matches the direction
 * expected by {@link aether.render} and {@link Aether.Provider}. */
export declare const createMockPair: () => [WorkerComms, MainComms];
//# sourceMappingURL=message.d.ts.map