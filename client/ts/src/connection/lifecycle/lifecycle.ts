// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, id, semver, TimeStamp } from "@synnaxlabs/x";

import {
  type Config,
  DEFAULT_STATUS,
  DEFAULT_STATUS_DETAILS,
  type ErrorStatus,
  type Info,
  type NonErrorStatus,
  type Reason,
  type Status,
  type StatusDetails,
} from "@/connection/status";
import {
  AuthError,
  ExpiredLicenseError,
  MissingCredentialsError,
  MissingLicenseError,
} from "@/errors";
import { license } from "@/license";
import type { status } from "@/status";

/**
 * Everything that can move the connection status. The client is the only
 * producer: check and retry events come from its own loop, the rest arrive as
 * outside facts through its notify inbox.
 */
export type Event =
  | { type: "check.started" }
  | { type: "check.success"; info: Info }
  /** `info` comes from a check without auth, when one answered. */
  | { type: "check.failure"; error: Error; attempt: number; info?: Info }
  | { type: "stream.live" }
  | { type: "stream.drop"; error?: Error }
  | { type: "stream.denied"; error?: Error }
  | { type: "auth.success"; session: string }
  | { type: "auth.failure"; error: Error }
  | { type: "epoch.advanced"; epoch: number }
  | { type: "retry.scheduled"; attempt: number; nextAt: TimeStamp }
  | { type: "retry.exhausted" }
  | { type: "retry.requested" }
  | { type: "credentials.replaced" }
  | { type: "closed" };

const CONNECTED = "Connected";
const CONNECTING = "Connecting";
const RECONNECTING = "Reconnecting";
const UNREACHABLE = "Cannot reach Core";
const INCOMPATIBLE = "Core version not supported";
const AUTH_FAILED = "Failed to authenticate";
const STREAM_DENIED =
  "Live updates are unavailable. This user cannot read the change channels.";

const licenseError = (state: Exclude<license.State, "ok">): Error =>
  state === "missing"
    ? new MissingLicenseError(license.STATE_MESSAGES.missing)
    : new ExpiredLicenseError(license.STATE_MESSAGES.expired);

/** The status a freshly constructed client starts in. */
export const createInitialStatus = (config: Config): Status => ({
  ...DEFAULT_STATUS,
  key: id.create(),
  name: config.name ?? DEFAULT_STATUS.name,
  time: TimeStamp.now(),
  variant: "loading",
  message: CONNECTING,
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
  description: string = "",
): NonErrorStatus => {
  const { reason: _, ...merged }: StatusDetails & { reason?: Reason } = {
    ...prev.details,
    ...details,
  };
  return { ...prev, variant, message, description, details: merged };
};

type ErrorCause =
  { reason: "unlicensed"; error: Error } | { reason: Exclude<Reason, "unlicensed"> };

const enterError = (
  prev: Status,
  message: string,
  details: Partial<StatusDetails> & ErrorCause,
  description: string = "",
): ErrorStatus => ({
  ...prev,
  variant: "error",
  message,
  description,
  details: { ...prev.details, ...details },
});

export const isSelfHealing = (reason: Reason): boolean => reason !== "auth";

// A 0.0 major.minor marks a development build, which pairs with anything.
const isDev = (version: string): boolean => version.startsWith("0.0.");

const isCompatible = (nodeVersion: string, clientVersion: string): boolean =>
  isDev(nodeVersion) ||
  isDev(clientVersion) ||
  semver.equal(clientVersion, nodeVersion, { patchIgnored: true });

// Exceeded skew clears only below this fraction of the threshold, so skew that hovers
// at the threshold does not flip the flag on every check.
const CLOCK_SKEW_CLEAR_FRACTION = 0.5;

const isSkewExceeded = (prev: Status, info: Info, config: Config): boolean => {
  const skew = info.clockSkew.abs();
  const { clockSkewThreshold } = config;
  const { clockSkewExceeded, clusterKey } = prev.details;
  if (clockSkewExceeded && clusterKey === info.clusterKey)
    return !skew.lessThan(clockSkewThreshold.mult(CLOCK_SKEW_CLEAR_FRACTION));
  // A slow round trip is not skew, so only skew beyond its error bound counts.
  return skew.greaterThan(clockSkewThreshold.add(info.roundTrip.div(2)));
};

const checkFacts = (
  prev: Status,
  info: Info,
  config: Config,
): Partial<StatusDetails> => ({
  clusterKey: info.clusterKey,
  nodeVersion: info.nodeVersion,
  nodeKey: info.nodeKey,
  // the check traverses the auth middleware, so a response is proof of auth
  authenticated: true,
  clockSkew: info.clockSkew,
  clockSkewExceeded: isSkewExceeded(prev, info, config),
  clientServerCompatible: isCompatible(info.nodeVersion, config.clientVersion),
  license: info.license,
});

// Connected, with live updates refused. A settled outcome: only a policy
// change lifts it, so the machine rests here instead of chasing the stream.
const enterStreamDenied = (
  prev: Status,
  details: Partial<StatusDetails> = {},
): Status =>
  enter(
    prev,
    "warning",
    CONNECTED,
    {
      ...details,
      streamDenied: true,
      streamLive: false,
      error: undefined,
      retry: null,
    },
    STREAM_DENIED,
  );

const reduceCheckSuccess = (prev: Status, info: Info, config: Config): Status => {
  const { clusterKey } = prev.details;
  if (clusterKey !== "" && info.clusterKey !== clusterKey)
    return reduceClusterReplaced(prev, info, config);
  const facts = checkFacts(prev, info, config);
  if (info.license !== "ok")
    return enterError(prev, license.STATE_MESSAGES[info.license], {
      ...facts,
      reason: "unlicensed",
      error: licenseError(info.license),
      retry: null,
    });
  // reachable but the stream is still dark: the client re-demands it and we
  // stay degraded until it reports live. A parked self-healing error must lift
  // out of error: an unreachable or unlicensed one drives a short circuit that
  // would starve the very stream reopen this state waits on.
  if (config.requiresStream && !prev.details.streamLive) {
    if (prev.details.streamDenied) return enterStreamDenied(prev, facts);
    if (prev.variant === "error" && isSelfHealing(prev.details.reason))
      return enter(prev, "loading", RECONNECTING, {
        ...facts,
        error: undefined,
        retry: null,
      });
    return update(prev, facts);
  }
  if (prev.variant === "success") return update(prev, { ...facts, retry: null });
  return enter(prev, "success", CONNECTED, {
    ...facts,
    error: undefined,
    retry: null,
  });
};

// The cluster answering the check is not the one previously contacted.
// Everything learned from the old cluster is void, so the machine returns to
// first contact: loading at epoch 0 with a dark stream.
const reduceClusterReplaced = (prev: Status, info: Info, config: Config): Status =>
  enter(prev, "loading", CONNECTING, {
    ...checkFacts(prev, info, config),
    streamLive: false,
    streamDenied: false,
    epoch: 0,
    error: undefined,
    retry: null,
  });

const reduceAuthFailure = (prev: Status, error: Error): Status => {
  // a later login with nothing to send must not hide why the credentials went away
  if (
    MissingCredentialsError.matches(error) &&
    prev.variant === "error" &&
    prev.details.reason === "auth"
  )
    return prev;
  return enterError(
    prev,
    AUTH_FAILED,
    { reason: "auth", error, authenticated: false, session: undefined, retry: null },
    error.message,
  );
};

const reduceCheckFailure = (
  prev: Status,
  error: Error,
  attempt: number,
  config: Config,
  info?: Info,
): Status => {
  if (info != null && !isCompatible(info.nodeVersion, config.clientVersion))
    return enterError(
      prev,
      INCOMPATIBLE,
      {
        ...checkFacts(prev, info, config),
        reason: "incompatible",
        error,
        authenticated: false,
        retry: null,
      },
      `Client ${config.clientVersion} cannot work with Core ${info.nodeVersion}.`,
    );
  if (AuthError.matches(error)) return reduceAuthFailure(prev, error);
  const escalating = prev.variant === "loading";
  if (escalating && attempt >= config.escalateAfter)
    return enterError(
      prev,
      UNREACHABLE,
      { reason: "unreachable", error },
      error.message,
    );
  // an unlicensed or incompatible Core that stops answering is unreachable
  if (
    prev.variant === "success" ||
    prev.variant === "warning" ||
    (prev.variant === "error" &&
      (prev.details.reason === "unlicensed" || prev.details.reason === "incompatible"))
  )
    return enter(prev, "loading", RECONNECTING, { error });
  return update(prev, { error });
};

const reduceStreamLive = (prev: Status): Status => {
  const parked = prev.variant === "error" && prev.details.reason !== "unreachable";
  // A stream opened under replaced credentials says nothing about the new ones.
  if (parked || !prev.details.authenticated)
    return update(prev, { streamLive: true, streamDenied: false });
  return enter(prev, "success", CONNECTED, {
    streamLive: true,
    streamDenied: false,
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
  return enterError(
    prev,
    UNREACHABLE,
    { reason: "unreachable", retry: null },
    prev.details.error?.message,
  );
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
      return update(
        reduceCheckFailure(prev, event.error, event.attempt, config, event.info),
        { checking: false },
      );
    case "stream.live":
      return reduceStreamLive(prev);
    case "stream.drop":
      return reduceStreamDrop(prev, event.error);
    case "stream.denied":
      return enterStreamDenied(prev);
    case "auth.success":
      // a login after a rejected one brings the connection back up
      if (prev.variant === "error" && prev.details.reason === "auth")
        return enter(prev, "loading", CONNECTING, {
          authenticated: true,
          session: event.session,
          error: undefined,
        });
      return update(prev, { authenticated: true, session: event.session });
    case "auth.failure":
      return reduceAuthFailure(prev, event.error);
    case "epoch.advanced":
      return update(prev, { epoch: event.epoch });
    case "retry.scheduled":
      return update(prev, { retry: { attempt: event.attempt, nextAt: event.nextAt } });
    case "retry.exhausted":
      return reduceRetryExhausted(prev);
    case "retry.requested":
      // deliberately does not clear auth: it rests until the user supplies new
      // credentials
      if (prev.variant !== "error" || !isSelfHealing(prev.details.reason)) return prev;
      return enter(prev, "loading", CONNECTING);
    case "credentials.replaced":
      // the new user may hold the permissions the old one lacked
      return enter(prev, "loading", CONNECTING, {
        error: undefined,
        streamDenied: false,
        authenticated: false,
        session: undefined,
      });
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

const materialChange = (prev: Status, next: Status): boolean =>
  !deep.equal(comparable(prev), comparable(next));

/**
 * Folds the event and stamps the status time when the result is worth publishing.
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
