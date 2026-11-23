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
  observe,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { DisconnectedError } from "@/errors";
import { license } from "@/license";
import { status } from "@/status";

export const REASONS = ["unreachable", "auth", "incompatible", "unlicensed"] as const;
export const reasonZ = z.enum(REASONS);
/** Why the connection is in the error variant. */
export type Reason = z.infer<typeof reasonZ>;

/** The fact vector beneath the connection's status variant. */
export const statusDetailsZ = z.object({
  error: z.instanceof(Error).optional(),
  authenticated: z.boolean(),
  // The key of the client's session, so a new session notifies observers. Never the
  // token: status details get logged.
  session: z.string().optional(),
  streamLive: z.boolean(),
  // The cluster refused the change stream. Unary calls the user is allowed to
  // make still work; live updates do not arrive.
  streamDenied: z.boolean(),
  epoch: z.number(),
  clusterKey: z.string(),
  clientVersion: z.string(),
  nodeVersion: z.string().optional(),
  // The key of the node the client connects to. Absent until the first check.
  nodeKey: z.number().optional(),
  clientServerCompatible: z.boolean(),
  clockSkew: TimeSpan.z,
  clockSkewExceeded: z.boolean(),
  retry: z.object({ attempt: z.number(), nextAt: TimeStamp.z }).nullable(),
  // Absent until the first check.
  license: license.stateZ.optional(),
  // A check is in flight right now. A process fact, not a judgment: the
  // variant holds its verdict while attempts run beneath it.
  checking: z.boolean(),
});
export interface StatusDetails extends z.infer<typeof statusDetailsZ> {}

/** Error-variant details: the same facts plus the reason the connection failed. */
export const errorStatusDetailsZ = z.discriminatedUnion("reason", [
  statusDetailsZ.extend({
    reason: z.literal("unlicensed"),
    error: z.instanceof(Error),
  }),
  statusDetailsZ.extend({ reason: reasonZ.exclude(["unlicensed"]) }),
]);
export type ErrorStatusDetails = z.infer<typeof errorStatusDetailsZ>;

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

export const DEFAULT_STATUS_DETAILS: StatusDetails = {
  authenticated: false,
  streamLive: false,
  streamDenied: false,
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
  labels: [],
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
   * Resets the retry backoff and checks immediately. Auth errors are not cleared: those
   * rest until the user supplies new credentials.
   */
  retryNow: () => void;
}

const isSettled = ({ variant }: Status): boolean =>
  variant === "success" ||
  variant === "warning" ||
  variant === "error" ||
  variant === "disabled";

/**
 * Resolves once the connection is usable: success, or warning when the cluster
 * refuses live updates. Rejects with the stored failure when it settles on an
 * error variant or is closed. The current status is evaluated first, so an
 * already-connected handle resolves without waiting.
 * @throws {Error} if the timeout elapses first.
 */
export const awaitConnected = async (
  handle: Handle,
  timeout?: CrudeTimeSpan,
): Promise<Status> => {
  const settled = await observe.until(handle, () => handle.status, isSettled, timeout);
  if (settled.variant !== "success" && settled.variant !== "warning")
    throw settled.details.error ?? new DisconnectedError(settled.message);
  return settled;
};

/** The facts a single connectivity check yields. */
export interface Info {
  clusterKey: string;
  nodeVersion: string;
  /** The key of the node that answered the check. */
  nodeKey: number;
  /** Skew measured across the check's round trip. */
  clockSkew: TimeSpan;
  /** The check's round trip. The skew is off by at most half of it. */
  roundTrip: TimeSpan;
  license: license.State;
}
