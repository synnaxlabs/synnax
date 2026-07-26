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
 * The connection status: a standard status whose details carry the connection's
 * facts. Renders anywhere a status does, with no translation.
 */
export const statusZ = status.statusZ({ details: detailsZ });
export type Status = status.Status<typeof detailsZ>;

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

/** The status of a client that does not exist. */
export const DEFAULT_STATUS: Status = {
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
  /** Human-readable cluster name for status messages. */
  name?: string;
  escalateAfter: number;
  clockSkewThreshold: TimeSpan;
  /** Whether success requires a live change stream. */
  requiresStream: boolean;
}

/**
 * The consumer-facing view of a client's connection. Read {@link Handle.status}
 * or subscribe via {@link Handle.onChange}; the client owns every transition.
 */
export interface Handle {
  /** The current connection status. */
  readonly status: Status;
  /** Subscribes to status changes. Returns a destructor that unsubscribes. */
  onChange: (callback: (status: Status) => void) => destructor.Destructor;
  /**
   * Resets the retry backoff and probes immediately. Auth and incompatibility
   * errors are not cleared: those rest until the user supplies something new.
   */
  retryNow: () => void;
}

const isSettled = ({ variant }: Status): boolean =>
  variant === "success" || variant === "error" || variant === "disabled";

/**
 * Resolves once the connection reaches success. Rejects with the stored failure
 * when it settles on an error variant or is closed. The current status is
 * evaluated first, so an already-connected handle resolves without waiting.
 * @throws {Error} if the timeout elapses first.
 */
export const awaitConnected = async (
  handle: Handle,
  timeout?: CrudeTimeSpan,
): Promise<Status> => {
  const settled = await observe.until(handle, () => handle.status, isSettled, timeout);
  if (settled.variant !== "success")
    throw settled.details.error ?? new DisconnectedError(settled.message);
  return settled;
};

/** The facts a single connectivity probe yields. */
export interface Info {
  clusterKey: string;
  nodeVersion?: string;
  /** Skew measured across the probe's round trip. */
  clockSkew: TimeSpan;
}

/**
 * Everything that can move the connection status. The client is the only
 * producer: the probe and retry events come from its prober, the stream events
 * from its change stream, the auth events from its login middleware.
 */
export type Event =
  | { type: "probe.success"; info: Info }
  | { type: "cluster.replaced"; info: Info }
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

/** The status a freshly constructed client starts in. */
export const createInitialStatus = (config: Config): Status => ({
  ...DEFAULT_STATUS,
  key: id.create(),
  name: config.name ?? DEFAULT_STATUS.name,
  time: TimeStamp.now(),
  variant: "loading",
  message: `Connecting to ${config.name ?? "cluster"}...`,
  details: { ...DEFAULT_DETAILS, clientVersion: config.clientVersion },
});

interface Changes extends Partial<Omit<Status, "details">> {
  details?: Partial<Details>;
}

const update = (prev: Status, changes: Changes): Status => ({
  ...prev,
  ...changes,
  details: { ...prev.details, ...changes.details },
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

const probeFacts = (info: Info, config: Config): Partial<Details> => ({
  clusterKey: info.clusterKey,
  nodeVersion: info.nodeVersion,
  // the probe traverses the auth middleware, so a response is proof of auth
  authenticated: true,
  clockSkew: info.clockSkew,
  clockSkewExceeded: info.clockSkew.abs().greaterThan(config.clockSkewThreshold),
  clientServerCompatible: isCompatible(info.nodeVersion, config.clientVersion),
});

const reduceProbeSuccess = (prev: Status, info: Info, config: Config): Status => {
  const facts = probeFacts(info, config);
  // reachable but the stream is still dark: the client re-demands it and we
  // stay degraded until it reports live
  if (config.requiresStream && !prev.details.streamLive)
    return update(prev, { details: facts });
  if (prev.variant === "success")
    return update(prev, { details: { ...facts, retry: null } });
  return update(prev, {
    variant: "success",
    message: connectedMessage(config),
    details: { ...facts, reason: undefined, error: undefined, retry: null },
  });
};

// The cluster answering at the connection's address is not the one previously
// contacted. Everything learned from the old cluster is void, so the machine
// returns to first contact: loading at epoch 0 with a dark stream.
const reduceClusterReplaced = (prev: Status, info: Info, config: Config): Status =>
  update(prev, {
    variant: "loading",
    message: `Connecting to ${config.name ?? "cluster"}...`,
    details: {
      ...probeFacts(info, config),
      streamLive: false,
      epoch: 0,
      reason: undefined,
      error: undefined,
      retry: null,
    },
  });

const reduceAuthFailure = (prev: Status, error: Error): Status =>
  update(prev, {
    variant: "error",
    message: error.message,
    details: { reason: "auth", error, authenticated: false, retry: null },
  });

const reduceProbeFailure = (
  prev: Status,
  error: Error,
  attempt: number,
  config: Config,
): Status => {
  if (AuthError.matches(error)) return reduceAuthFailure(prev, error);
  const escalating = prev.variant === "loading" || prev.variant === "warning";
  if (escalating && attempt >= config.escalateAfter)
    return update(prev, {
      variant: "error",
      message: error.message ?? UNREACHABLE,
      details: { reason: "unreachable", error },
    });
  if (prev.variant === "success")
    return update(prev, {
      variant: "warning",
      message: RECONNECTING,
      details: { error },
    });
  return update(prev, { details: { error } });
};

const reduceStreamLive = (prev: Status, config: Config): Status => {
  const parked = prev.variant === "error" && prev.details.reason !== "unreachable";
  if (parked) return update(prev, { details: { streamLive: true } });
  return update(prev, {
    variant: "success",
    message: connectedMessage(config),
    details: { streamLive: true, reason: undefined, error: undefined, retry: null },
  });
};

const reduceStreamDrop = (prev: Status, error?: Error): Status => {
  if (prev.variant !== "success")
    return update(prev, { details: { streamLive: false } });
  return update(prev, {
    variant: "warning",
    message: RECONNECTING,
    details: { streamLive: false, error },
  });
};

const reduceRetryExhausted = (prev: Status): Status => {
  if (prev.variant !== "loading" && prev.variant !== "warning")
    return update(prev, { details: { retry: null } });
  return update(prev, {
    variant: "error",
    message: prev.details.error?.message ?? UNREACHABLE,
    details: { reason: "unreachable", retry: null },
  });
};

/**
 * Folds an event into the connection status. Pure: every transition in the
 * lifecycle lives here and nowhere else. `disabled` is terminal.
 */
export const reduce = (prev: Status, event: Event, config: Config): Status => {
  if (prev.variant === "disabled") return prev;
  switch (event.type) {
    case "probe.success":
      return reduceProbeSuccess(prev, event.info, config);
    case "cluster.replaced":
      return reduceClusterReplaced(prev, event.info, config);
    case "probe.failure":
      return reduceProbeFailure(prev, event.error, event.attempt, config);
    case "stream.live":
      return reduceStreamLive(prev, config);
    case "stream.drop":
      return reduceStreamDrop(prev, event.error);
    case "auth.success":
      return update(prev, { details: { authenticated: true } });
    case "auth.failure":
      return reduceAuthFailure(prev, event.error);
    case "epoch.advanced":
      return update(prev, { details: { epoch: event.epoch } });
    case "retry.scheduled":
      return update(prev, {
        details: { retry: { attempt: event.attempt, nextAt: event.nextAt } },
      });
    case "retry.exhausted":
      return reduceRetryExhausted(prev);
    case "retry.requested":
      // deliberately does not clear auth or incompatibility: those rest until
      // the user supplies something new
      if (prev.variant !== "error" || prev.details.reason !== "unreachable")
        return prev;
      return update(prev, {
        variant: "loading",
        message: CONNECTING,
        details: { reason: undefined },
      });
    case "credentials.replaced":
      return update(prev, {
        variant: "loading",
        message: CONNECTING,
        details: { reason: undefined, error: undefined },
      });
    case "closed":
      return update(prev, {
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
export const materialChange = (prev: Status, next: Status): boolean =>
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
 * Folds the event and stamps the status time when the result is worth
 * publishing.
 * @returns the next status and whether observers should be notified.
 */
export const advance = (
  prev: Status,
  event: Event,
  config: Config,
): [Status, boolean] => {
  const next = reduce(prev, event, config);
  if (!materialChange(prev, next)) return [next, false];
  return [{ ...next, time: TimeStamp.now() }, true];
};
