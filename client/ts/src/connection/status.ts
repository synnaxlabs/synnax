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
  deep,
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

/** The fact vector beneath the connection's status variant. */
export const statusDetailsZ = z.object({
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
  // A check is in flight right now. A process fact, not a judgment: the
  // variant holds its verdict while attempts run beneath it.
  checking: z.boolean(),
});
export interface StatusDetails extends z.infer<typeof statusDetailsZ> {}

/** Error-variant details: the same facts plus the reason the connection failed. */
export const errorStatusDetailsZ = statusDetailsZ.extend({ reason: reasonZ });
export interface ErrorStatusDetails extends z.infer<typeof errorStatusDetailsZ> {}

export const nonErrorVariantZ = z.enum([
  "success",
  "info",
  "warning",
  "loading",
  "disabled",
]);

/**
 * The connection status: a standard status whose details carry the connection's
 * facts. Renders anywhere a status does, with no translation. Discriminated on
 * variant: `reason` exists only on the error member.
 */
export const statusZ = z.discriminatedUnion("variant", [
  status.statusZ({ v: status.errorVariantZ, details: errorStatusDetailsZ }),
  status.statusZ({ v: nonErrorVariantZ, details: statusDetailsZ }),
]);
export type ErrorStatus = status.Status<
  typeof errorStatusDetailsZ,
  typeof status.errorVariantZ
>;
export type NonErrorStatus = status.Status<
  typeof statusDetailsZ,
  typeof nonErrorVariantZ
>;
export type Status = ErrorStatus | NonErrorStatus;

const DEFAULT_STATUS_DETAILS: StatusDetails = {
  authenticated: false,
  streamLive: false,
  epoch: 0,
  clusterKey: "",
  clientVersion: __VERSION__,
  clientServerCompatible: false,
  clockSkew: TimeSpan.ZERO,
  clockSkewExceeded: false,
  retry: null,
  checking: false,
};

/** The status of a client that does not exist. */
export const DEFAULT_STATUS: Status = {
  key: "connection",
  name: "Connection",
  variant: "disabled",
  message: "Disconnected",
  description: "",
  time: TimeStamp.ZERO,
  details: DEFAULT_STATUS_DETAILS,
};

/** Consecutive check failures before escalating to error(unreachable). */
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
   * Resets the retry backoff and checks immediately. Auth and incompatibility
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

/** The facts a single connectivity check yields. */
export interface Info {
  clusterKey: string;
  nodeVersion?: string;
  /** Skew measured across the check's round trip. */
  clockSkew: TimeSpan;
}

/**
 * Everything that can move the connection status. The client is the only
 * producer: check and retry events come from its own loop, the rest arrive as
 * outside facts through its notify inbox.
 */
export type Event =
  | { type: "check.started" }
  | { type: "check.success"; info: Info }
  | { type: "check.failure"; error: Error; attempt: number }
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

const CONNECTING = "Connecting";
const RECONNECTING = "Reconnecting";
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
  message: `Connecting to ${config.name ?? "cluster"}`,
  details: { ...DEFAULT_STATUS_DETAILS, clientVersion: config.clientVersion },
});

// Merges facts without touching variant or reason, so the union member holds.
const update = <S extends Status>(prev: S, details: Partial<StatusDetails>): S => ({
  ...prev,
  details: { ...prev.details, ...details },
});

/** Moves to a non-error variant, structurally dropping any parked reason. */
const enter = (
  prev: Status,
  variant: Exclude<status.Variant, "error">,
  message: string,
  details: Partial<StatusDetails> = {},
): NonErrorStatus => {
  const { reason: _, ...merged }: StatusDetails & { reason?: Reason } = {
    ...prev.details,
    ...details,
  };
  return { ...prev, variant, message, details: merged };
};

const enterError = (
  prev: Status,
  message: string,
  reason: Reason,
  details: Partial<StatusDetails> = {},
): ErrorStatus => ({
  ...prev,
  variant: "error",
  message,
  details: { ...prev.details, ...details, reason },
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

const checkFacts = (info: Info, config: Config): Partial<StatusDetails> => ({
  clusterKey: info.clusterKey,
  nodeVersion: info.nodeVersion,
  // the check traverses the auth middleware, so a response is proof of auth
  authenticated: true,
  clockSkew: info.clockSkew,
  clockSkewExceeded: info.clockSkew.abs().greaterThan(config.clockSkewThreshold),
  clientServerCompatible: isCompatible(info.nodeVersion, config.clientVersion),
});

const reduceCheckSuccess = (prev: Status, info: Info, config: Config): Status => {
  const { clusterKey } = prev.details;
  if (clusterKey !== "" && info.clusterKey !== clusterKey)
    return reduceClusterReplaced(prev, info, config);
  const facts = checkFacts(info, config);
  // reachable but the stream is still dark: the client re-demands it and we
  // stay degraded until it reports live. A parked unreachable error must
  // lift out of error: the short circuit it drives would starve the very
  // stream reopen this state waits on.
  if (config.requiresStream && !prev.details.streamLive) {
    if (prev.variant === "error" && prev.details.reason === "unreachable")
      return enter(prev, "loading", RECONNECTING, {
        ...facts,
        error: undefined,
        retry: null,
      });
    return update(prev, facts);
  }
  if (prev.variant === "success") return update(prev, { ...facts, retry: null });
  return enter(prev, "success", connectedMessage(config), {
    ...facts,
    error: undefined,
    retry: null,
  });
};

// The cluster answering the check is not the one previously contacted.
// Everything learned from the old cluster is void, so the machine returns to
// first contact: loading at epoch 0 with a dark stream.
const reduceClusterReplaced = (prev: Status, info: Info, config: Config): Status =>
  enter(prev, "loading", `Connecting to ${config.name ?? "cluster"}`, {
    ...checkFacts(info, config),
    streamLive: false,
    epoch: 0,
    error: undefined,
    retry: null,
  });

const reduceAuthFailure = (prev: Status, error: Error): Status =>
  enterError(prev, error.message, "auth", {
    error,
    authenticated: false,
    retry: null,
  });

const reduceCheckFailure = (
  prev: Status,
  error: Error,
  attempt: number,
  config: Config,
): Status => {
  if (AuthError.matches(error)) return reduceAuthFailure(prev, error);
  const escalating = prev.variant === "loading";
  if (escalating && attempt >= config.escalateAfter)
    return enterError(prev, error.message ?? UNREACHABLE, "unreachable", { error });
  if (prev.variant === "success")
    return enter(prev, "loading", RECONNECTING, { error });
  return update(prev, { error });
};

const reduceStreamLive = (prev: Status, config: Config): Status => {
  const parked = prev.variant === "error" && prev.details.reason !== "unreachable";
  if (parked) return update(prev, { streamLive: true });
  return enter(prev, "success", connectedMessage(config), {
    streamLive: true,
    error: undefined,
    retry: null,
  });
};

const reduceStreamDrop = (prev: Status, error?: Error): Status => {
  if (prev.variant !== "success") return update(prev, { streamLive: false });
  return enter(prev, "loading", RECONNECTING, { streamLive: false, error });
};

const reduceRetryExhausted = (prev: Status): Status => {
  if (prev.variant !== "loading") return update(prev, { retry: null });
  return enterError(prev, prev.details.error?.message ?? UNREACHABLE, "unreachable", {
    retry: null,
  });
};

/**
 * Folds an event into the connection status. Pure: every transition in the
 * lifecycle lives here and nowhere else. `disabled` is terminal.
 */
export const reduce = (prev: Status, event: Event, config: Config): Status => {
  if (prev.variant === "disabled") return prev;
  switch (event.type) {
    case "check.started":
      return update(prev, { checking: true });
    case "check.success":
      return update(reduceCheckSuccess(prev, event.info, config), { checking: false });
    case "check.failure":
      return update(reduceCheckFailure(prev, event.error, event.attempt, config), {
        checking: false,
      });
    case "stream.live":
      return reduceStreamLive(prev, config);
    case "stream.drop":
      return reduceStreamDrop(prev, event.error);
    case "auth.success":
      return update(prev, { authenticated: true });
    case "auth.failure":
      return reduceAuthFailure(prev, event.error);
    case "epoch.advanced":
      return update(prev, { epoch: event.epoch });
    case "retry.scheduled":
      return update(prev, { retry: { attempt: event.attempt, nextAt: event.nextAt } });
    case "retry.exhausted":
      return reduceRetryExhausted(prev);
    case "retry.requested":
      // deliberately does not clear auth or incompatibility: those rest until
      // the user supplies something new
      if (prev.variant !== "error" || prev.details.reason !== "unreachable")
        return prev;
      return enter(prev, "loading", CONNECTING);
    case "credentials.replaced":
      return enter(prev, "loading", CONNECTING, { error: undefined });
    case "closed":
      return enter(prev, "disabled", "Closed", {
        streamLive: false,
        retry: null,
        checking: false,
      });
  }
};

// Fields that legitimately churn without meaning anything: the stamp moves on
// every advance and raw skew jitters on every successful check. Everything
// else is material, so new fields notify by default.
const comparable = ({ time: _, ...rest }: Status): unknown => ({
  ...rest,
  details: { ...rest.details, clockSkew: undefined },
});

/** Whether the change is worth notifying observers about. */
export const materialChange = (prev: Status, next: Status): boolean =>
  !deep.equal(comparable(prev), comparable(next));

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
