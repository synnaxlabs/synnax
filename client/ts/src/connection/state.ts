// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeTimeSpan,
  type destructor,
  id,
  migrate,
  observe,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { AuthError, DisconnectedError } from "@/errors";
import { status } from "@/status";

export const REASONS = ["unreachable", "auth", "incompatible"] as const;
export const reasonZ = z.enum(REASONS);
/** Why the connection is in the error variant. */
export type Reason = z.infer<typeof reasonZ>;

/**
 * The fact vector beneath the connection's status variant. `reason` is present
 * iff the variant is "error".
 */
export const detailsZ = z.object({
  reason: reasonZ.optional(),
  error: z.instanceof(Error).optional(),
  authenticated: z.boolean(),
  streamLive: z.boolean(),
  epoch: z.number(),
  clusterKey: z.string(),
  clientVersion: z.string(),
  nodeVersion: z.string().optional(),
  clientServerCompatible: z.boolean(),
  clockSkew: TimeSpan.z,
  clockSkewExceeded: z.boolean(),
  retry: z.object({ attempt: z.number(), nextAt: TimeStamp.z }).nullable(),
});
export interface Details extends z.infer<typeof detailsZ> {}

/**
 * The connection state: a standard status whose details carry the connection's
 * facts. Renders anywhere a status does, with no translation.
 */
export const stateZ = status.statusZ({ details: detailsZ });
export type State = status.Status<typeof detailsZ>;

export const DEFAULT_DETAILS: Details = {
  authenticated: false,
  streamLive: false,
  epoch: 0,
  clusterKey: "",
  clientVersion: __VERSION__,
  clientServerCompatible: false,
  clockSkew: TimeSpan.ZERO,
  clockSkewExceeded: false,
  retry: null,
};

/** The state of a client that does not exist. */
export const DEFAULT_STATE: State = {
  key: "connection",
  name: "Connection",
  variant: "disabled",
  message: "Disconnected",
  description: "",
  time: TimeStamp.ZERO,
  details: DEFAULT_DETAILS,
};

/** Consecutive probe failures before escalating to error(unreachable). */
export const DEFAULT_ESCALATE_AFTER = 4;

export interface Config {
  clientVersion: string;
  /** Human-readable cluster name for state messages. */
  name?: string;
  escalateAfter: number;
  clockSkewThreshold: TimeSpan;
  /** Whether success requires a live change stream. */
  requiresStream: boolean;
}

/**
 * The consumer-facing view of a client's connection. Read {@link Handle.state}
 * or subscribe via {@link Handle.onChange}; the client owns every transition.
 */
export interface Handle {
  /** The current connection state. */
  readonly state: State;
  /** Subscribes to state changes. Returns a destructor that unsubscribes. */
  onChange: (callback: (state: State) => void) => destructor.Destructor;
  /**
   * Resets the retry backoff and probes immediately. Auth and incompatibility
   * errors are not cleared: those rest until the user supplies something new.
   */
  retryNow: () => void;
}

const isSettled = ({ variant }: State): boolean =>
  variant === "success" || variant === "error" || variant === "disabled";

/**
 * Resolves once the connection reaches success. Rejects with the stored failure
 * when it settles on an error variant or is closed. The current state is
 * evaluated first, so an already-connected handle resolves without waiting.
 * @throws {Error} if the timeout elapses first.
 */
export const awaitConnected = async (
  handle: Handle,
  timeout?: CrudeTimeSpan,
): Promise<State> => {
  const state = await observe.until(handle, () => handle.state, isSettled, timeout);
  if (state.variant !== "success")
    throw state.details.error ?? new DisconnectedError(state.message);
  return state;
};

/** The facts a single connectivity probe yields. */
export interface Info {
  clusterKey: string;
  nodeVersion?: string;
  /** Skew measured across the probe's round trip. */
  clockSkew: TimeSpan;
}

/**
 * Everything that can move the connection state. The client is the only
 * producer: the probe and retry events come from its prober, the stream events
 * from its change stream, the auth events from its login middleware.
 */
export type Event =
  | { type: "probe.success"; info: Info }
  | { type: "probe.failure"; error: Error; attempt: number }
  | { type: "stream.live" }
  | { type: "stream.drop"; error?: Error }
  | { type: "auth.success" }
  | { type: "auth.failure"; error: Error }
  | { type: "epoch.advanced"; epoch: number }
  | { type: "retry.scheduled"; attempt: number; nextAt: TimeStamp }
  | { type: "retry.exhausted" }
  | { type: "retry.requested" }
  | { type: "credentials.replaced" }
  | { type: "closed" };

const CONNECTING = "Connecting...";
const RECONNECTING = "Reconnecting...";
const UNREACHABLE = "Cannot reach cluster";

const connectedMessage = ({ name }: Config): string =>
  `Connected to ${name ?? "cluster"}`;

/** The state a freshly constructed client starts in. */
export const createInitialState = (config: Config): State => ({
  ...DEFAULT_STATE,
  key: id.create(),
  name: config.name ?? DEFAULT_STATE.name,
  time: TimeStamp.now(),
  variant: "loading",
  message: `Connecting to ${config.name ?? "cluster"}...`,
  details: { ...DEFAULT_DETAILS, clientVersion: config.clientVersion },
});

interface Changes extends Partial<Omit<State, "details">> {
  details?: Partial<Details>;
}

const update = (state: State, changes: Changes): State => ({
  ...state,
  ...changes,
  details: { ...state.details, ...changes.details },
});

const isCompatible = (
  nodeVersion: string | undefined,
  clientVersion: string,
): boolean =>
  nodeVersion != null &&
  migrate.versionsEqual(clientVersion, nodeVersion, {
    checkMajor: true,
    checkMinor: true,
    checkPatch: false,
  });

const reduceProbeSuccess = (state: State, info: Info, config: Config): State => {
  const facts: Partial<Details> = {
    clusterKey: info.clusterKey,
    nodeVersion: info.nodeVersion,
    // the probe traverses the auth middleware, so a response is proof of auth
    authenticated: true,
    clockSkew: info.clockSkew,
    clockSkewExceeded: info.clockSkew.abs().greaterThan(config.clockSkewThreshold),
    clientServerCompatible: isCompatible(info.nodeVersion, config.clientVersion),
  };
  // reachable but the stream is still dark: the client re-demands it and we
  // stay degraded until it reports live
  if (config.requiresStream && !state.details.streamLive)
    return update(state, { details: facts });
  if (state.variant === "success")
    return update(state, { details: { ...facts, retry: null } });
  return update(state, {
    variant: "success",
    message: connectedMessage(config),
    details: { ...facts, reason: undefined, error: undefined, retry: null },
  });
};

const reduceAuthFailure = (state: State, error: Error): State =>
  update(state, {
    variant: "error",
    message: error.message,
    details: { reason: "auth", error, authenticated: false, retry: null },
  });

const reduceProbeFailure = (
  state: State,
  error: Error,
  attempt: number,
  config: Config,
): State => {
  if (AuthError.matches(error)) return reduceAuthFailure(state, error);
  const escalating = state.variant === "loading" || state.variant === "warning";
  if (escalating && attempt >= config.escalateAfter)
    return update(state, {
      variant: "error",
      message: error.message ?? UNREACHABLE,
      details: { reason: "unreachable", error },
    });
  if (state.variant === "success")
    return update(state, {
      variant: "warning",
      message: RECONNECTING,
      details: { error },
    });
  return update(state, { details: { error } });
};

const reduceStreamLive = (state: State, config: Config): State => {
  const parked = state.variant === "error" && state.details.reason !== "unreachable";
  if (parked) return update(state, { details: { streamLive: true } });
  return update(state, {
    variant: "success",
    message: connectedMessage(config),
    details: { streamLive: true, reason: undefined, error: undefined, retry: null },
  });
};

const reduceStreamDrop = (state: State, error?: Error): State => {
  if (state.variant !== "success")
    return update(state, { details: { streamLive: false } });
  return update(state, {
    variant: "warning",
    message: RECONNECTING,
    details: { streamLive: false, error },
  });
};

const reduceRetryExhausted = (state: State): State => {
  if (state.variant !== "loading" && state.variant !== "warning")
    return update(state, { details: { retry: null } });
  return update(state, {
    variant: "error",
    message: state.details.error?.message ?? UNREACHABLE,
    details: { reason: "unreachable", retry: null },
  });
};

/**
 * Folds an event into the connection state. Pure: every transition in the
 * lifecycle lives here and nowhere else. `disabled` is terminal.
 */
export const reduce = (state: State, event: Event, config: Config): State => {
  if (state.variant === "disabled") return state;
  switch (event.type) {
    case "probe.success":
      return reduceProbeSuccess(state, event.info, config);
    case "probe.failure":
      return reduceProbeFailure(state, event.error, event.attempt, config);
    case "stream.live":
      return reduceStreamLive(state, config);
    case "stream.drop":
      return reduceStreamDrop(state, event.error);
    case "auth.success":
      return update(state, { details: { authenticated: true } });
    case "auth.failure":
      return reduceAuthFailure(state, event.error);
    case "epoch.advanced":
      return update(state, { details: { epoch: event.epoch } });
    case "retry.scheduled":
      return update(state, {
        details: { retry: { attempt: event.attempt, nextAt: event.nextAt } },
      });
    case "retry.exhausted":
      return reduceRetryExhausted(state);
    case "retry.requested":
      // deliberately does not clear auth or incompatibility: those rest until
      // the user supplies something new
      if (state.variant !== "error" || state.details.reason !== "unreachable")
        return state;
      return update(state, {
        variant: "loading",
        message: CONNECTING,
        details: { reason: undefined },
      });
    case "credentials.replaced":
      return update(state, {
        variant: "loading",
        message: CONNECTING,
        details: { reason: undefined, error: undefined },
      });
    case "closed":
      return update(state, {
        variant: "disabled",
        message: "Closed",
        details: { reason: undefined, streamLive: false, retry: null },
      });
  }
};

/**
 * Whether the change is worth notifying observers about. Error objects, raw
 * skew, and the next-probe timestamp move silently.
 */
export const materialChange = (prev: State, next: State): boolean =>
  next.variant !== prev.variant ||
  next.message !== prev.message ||
  next.details.reason !== prev.details.reason ||
  next.details.authenticated !== prev.details.authenticated ||
  next.details.streamLive !== prev.details.streamLive ||
  next.details.epoch !== prev.details.epoch ||
  next.details.clusterKey !== prev.details.clusterKey ||
  next.details.clientServerCompatible !== prev.details.clientServerCompatible ||
  next.details.clockSkewExceeded !== prev.details.clockSkewExceeded ||
  next.details.retry?.attempt !== prev.details.retry?.attempt;

/**
 * Folds the event and stamps the state's time when the result is worth
 * publishing.
 * @returns the next state and whether observers should be notified.
 */
export const advance = (
  state: State,
  event: Event,
  config: Config,
): [State, boolean] => {
  const next = reduce(state, event, config);
  if (!materialChange(state, next)) return [next, false];
  return [{ ...next, time: TimeStamp.now() }, true];
};
