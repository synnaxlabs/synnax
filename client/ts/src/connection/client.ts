// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, type UnaryClient } from "@synnaxlabs/freighter";
import {
  breaker,
  ClockSkewCalculator,
  type CrudeTimeSpan,
  type destructor,
  errors,
  migrate,
  observe,
  sync,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { auth } from "@/auth";
import {
  advance,
  awaitConnected,
  type Config,
  createInitialStatus,
  DEFAULT_ESCALATE_AFTER,
  type Event,
  type Handle,
  type Info,
  reduce,
  type Status,
} from "@/connection/status";
import { DisconnectedError } from "@/errors";

export const CHECK_ENDPOINT = "/connectivity/check";

const checkResZ = z.object({
  clusterKey: z.string(),
  nodeVersion: z.string().optional(),
  nodeTime: TimeStamp.z,
});

/**
 * Runs a single connectivity probe, measuring clock skew across the round trip.
 * The transport must carry the full middleware chain, auth included, so that a
 * response proves authentication.
 * @throws {AuthError} if the cluster rejects the client's credentials.
 * @throws {Unreachable} if the cluster cannot be reached.
 */
export const probe = async (unary: UnaryClient): Promise<Info> => {
  const skew = new ClockSkewCalculator();
  skew.start();
  const res = await unary.send(CHECK_ENDPOINT, undefined, z.void(), checkResZ);
  skew.end(res.nodeTime);
  return {
    clusterKey: res.clusterKey,
    nodeVersion: res.nodeVersion,
    clockSkew: skew.skew,
  };
};

/**
 * Runs a one-shot connectivity check against the given transport and folds the
 * result into a status. Never throws: failures land in the returned status.
 */
export const check = async (
  unary: UnaryClient,
  config: Partial<Config> = {},
): Promise<Status> => {
  const cfg: Config = {
    clientVersion: __VERSION__,
    escalateAfter: 1,
    clockSkewThreshold: TimeSpan.seconds(1),
    requiresStream: false,
    ...config,
  };
  const initial = createInitialStatus(cfg);
  try {
    const info = await probe(unary);
    return reduce(initial, { type: "probe.success", info }, cfg);
  } catch (err) {
    const error = errors.fromUnknown(err);
    return reduce(initial, { type: "probe.failure", error, attempt: 1 }, cfg);
  }
};

/** How hard the probe loop is working. */
export type Mode = "probing" | "heartbeat" | "idle";

/** The mode a given connection status calls for. */
export const modeFor = ({ variant, details }: Status): Mode => {
  switch (variant) {
    case "success":
      return "heartbeat";
    case "disabled":
      return "idle";
    case "error":
      // unreachable keeps probing beneath the error and self-heals; auth and
      // incompatibility rest until the user acts
      return details.reason === "unreachable" ? "probing" : "idle";
    default:
      return "probing";
  }
};

export const DEFAULT_RETRY: breaker.Config = {
  baseInterval: TimeSpan.seconds(1),
  // probes are one cheap request: a low cap keeps a returned cluster from
  // going unnoticed while backoff is escalated
  maxInterval: TimeSpan.seconds(5),
  maxRetries: Infinity,
  scale: 2,
  jitter: 0.25,
};

/**
 * The outside facts that can move the connection. Everything else that moves it
 * (probe results, retry scheduling) the client observes itself.
 */
export type FactEvent = Extract<
  Event,
  {
    type:
      | "auth.success"
      | "auth.failure"
      | "stream.live"
      | "stream.drop"
      | "credentials.replaced"
      | "epoch.advanced";
  }
>;

export interface ClientParams {
  /**
   * Probe transport. Must carry the full middleware chain, auth included, so
   * that a probe response proves authentication.
   */
  unary: UnaryClient;
  /** Cluster address, used in error messages. */
  address: string;
  /** Defaults to the library version. */
  clientVersion?: string;
  /** Human-readable cluster name for status messages. */
  name?: string;
  /** Consecutive probe failures before escalating to error(unreachable). */
  escalateAfter?: number;
  clockSkewThreshold?: TimeSpan;
  /** Whether success requires a live change stream. */
  requiresStream?: boolean;
  /**
   * Degraded-probe policy. `maxRetries` defaults to Infinity (never give up);
   * a finite value parks the loop once exhausted, until {@link Client.retryNow}.
   */
  retry?: breaker.Config;
  /** Probe cadence while connected. Defaults to 30 seconds. */
  heartbeatInterval?: CrudeTimeSpan;
  /** Levers on the change stream, pulled when transitions demand it. */
  stream?: {
    /** Tears down cached state after a cluster replacement. */
    reset: () => Promise<void>;
    /** (Re)demands a live change stream. */
    ensure: () => Promise<void>;
  };
  /** Receives errors with no caller to throw to. */
  onInternalError?: (error: Error) => void;
}

/**
 * Owns a cluster connection's status: probes on a cadence, backs off while
 * degraded, folds outside facts ({@link Client.notify}) into the status, and
 * notifies observers on material changes. The probe loop starts on
 * construction, deferred one microtask so callers can finish synchronous
 * wiring (middleware installation) first.
 */
export class Client implements Handle {
  private readonly unary: UnaryClient;
  private readonly address: string;
  private readonly config: Config;
  private readonly retry: breaker.Config;
  private readonly heartbeatInterval: TimeSpan;
  private readonly stream?: ClientParams["stream"];
  private readonly onInternalError?: (error: Error) => void;
  private readonly observer = new observe.Observer<Status>();
  private readonly notifier = new sync.Notifier();
  private readonly loop: Promise<void>;
  private current: Status;
  private mode: Mode = "probing";
  private closed = false;
  private attempts = 0;
  private generation = 0;
  private brk: breaker.Breaker | null = null;
  private versionWarned = false;

  constructor(params: ClientParams) {
    this.unary = params.unary;
    this.address = params.address;
    this.config = {
      clientVersion: params.clientVersion ?? __VERSION__,
      name: params.name,
      escalateAfter: params.escalateAfter ?? DEFAULT_ESCALATE_AFTER,
      clockSkewThreshold: params.clockSkewThreshold ?? TimeSpan.seconds(1),
      requiresStream: params.requiresStream ?? false,
    };
    this.retry = { ...DEFAULT_RETRY, ...params.retry };
    this.heartbeatInterval = new TimeSpan(
      params.heartbeatInterval ?? TimeSpan.seconds(30),
    );
    this.stream = params.stream;
    this.onInternalError = params.onInternalError;
    this.current = createInitialStatus(this.config);
    this.loop = this.run().catch((err: unknown) =>
      this.onInternalError?.(new Error("connection probe loop failed", { cause: err })),
    );
  }

  /** The current connection status. */
  get status(): Status {
    return this.current;
  }

  /** Subscribes to material status changes. Returns an unsubscribe destructor. */
  onChange(callback: (status: Status) => void): destructor.Destructor {
    return this.observer.onChange(callback);
  }

  /**
   * Resets the retry backoff and probes immediately. Auth and incompatibility
   * errors are not cleared: those rest until the user supplies something new.
   */
  retryNow(): void {
    this.dispatch({ type: "retry.requested" });
    this.wake();
  }

  /**
   * Resolves once the connection reaches success. Rejects with the stored
   * failure when it settles on an error variant or is closed. An
   * already-connected client resolves without waiting.
   * @throws {Error} if the timeout elapses first.
   */
  async connect(timeout?: CrudeTimeSpan): Promise<Status> {
    return await awaitConnected(this, timeout);
  }

  /** Folds an outside fact into the connection status. */
  notify(event: FactEvent): void {
    this.dispatch(event);
    // a reopened stream may lead to a replaced cluster, and new credentials
    // deserve an immediate verdict: both probe right away
    if (event.type === "stream.live" || event.type === "credentials.replaced")
      this.wake();
  }

  /**
   * Rejects unary requests instantly while the cluster is known unreachable,
   * instead of burning the transport's retry budget per call. The probe and
   * login targets are exempt so the connection can heal.
   */
  middleware(): Middleware {
    const EXEMPT = [CHECK_ENDPOINT, auth.LOGIN_ENDPOINT];
    return async (ctx, next) => {
      const { variant, details } = this.current;
      if (
        variant === "error" &&
        details.reason === "unreachable" &&
        !EXEMPT.some((target) => ctx.target.endsWith(target))
      )
        throw new DisconnectedError(`Cannot reach cluster at ${this.address}`);
      return await next(ctx);
    };
  }

  /** Stops the probe loop and settles the status on disabled. Terminal. */
  async close(): Promise<void> {
    if (this.closed) return;
    this.dispatch({ type: "closed" });
    this.closed = true;
    this.notifier.notify();
    await this.loop;
  }

  /**
   * Folds the event into the status, notifies observers, and applies the
   * transition's side effects.
   */
  private dispatch(event: Event): void {
    if (this.closed) return;
    const prev = this.current;
    const [next, changed] = advance(prev, event, this.config);
    this.current = next;
    if (changed) this.observer.notify(next);
    // a new cluster at the connection's address voids everything learned from
    // the old one: drop cached state, then bring the stream back up
    if (
      prev.details.clusterKey !== "" &&
      next.details.clusterKey !== prev.details.clusterKey
    )
      this.stream
        ?.reset()
        .then(async () => await this.stream?.ensure())
        .catch((err: unknown) =>
          this.onInternalError?.(
            new Error("failed to reset cache after cluster replacement", {
              cause: err,
            }),
          ),
        );
    this.setMode(modeFor(next));
  }

  /** Discards in-flight probes and wakes the loop for an immediate probe. */
  private wake(): void {
    if (this.closed) return;
    this.generation += 1;
    this.attempts = 0;
    this.brk?.reset();
    this.notifier.notify();
  }

  private setMode(mode: Mode): void {
    if (this.closed || this.mode === mode) return;
    this.mode = mode;
    // in-flight probes were aimed at the state the old mode ran under; their
    // late results must not degrade the new one
    this.generation += 1;
    if (mode !== "probing") this.attempts = 0;
    this.brk?.reset();
    this.notifier.notify();
  }

  private async runProbe(): Promise<void> {
    const generation = this.generation;
    const prevKey = this.current.details.clusterKey;
    this.dispatch({ type: "probe.started" });
    try {
      const info = await probe(this.unary);
      if (generation !== this.generation || this.closed) return;
      this.attempts = 0;
      this.dispatch({ type: "probe.success", info });
      this.warn(this.current.details);
      const replaced = prevKey !== "" && info.clusterKey !== prevKey;
      // reachable but the stream is still dark: re-demand it, in case its own
      // retry budget was exhausted. Replacement already brings it up.
      if (!replaced && this.config.requiresStream && !this.current.details.streamLive)
        this.stream
          ?.ensure()
          .catch((err: unknown) =>
            this.onInternalError?.(
              new Error("failed to bring up change stream", { cause: err }),
            ),
          );
    } catch (err) {
      if (generation !== this.generation || this.closed) return;
      this.attempts += 1;
      this.dispatch({
        type: "probe.failure",
        error: errors.fromUnknown(err),
        attempt: this.attempts,
      });
    }
  }

  private async run(): Promise<void> {
    // defer the first probe so the caller can finish synchronous wiring
    // (middleware installation) after construction
    await Promise.resolve();
    const brk = new breaker.Breaker({
      ...this.retry,
      sleepFn: async (duration) => {
        this.dispatch({
          type: "retry.scheduled",
          attempt: this.attempts,
          nextAt: TimeStamp.now().add(duration),
        });
        await this.notifier.wait(duration);
      },
    });
    this.brk = brk;
    while (!this.closed)
      if (this.mode === "probing") {
        await this.runProbe();
        if (this.closed) return;
        if (this.mode !== "probing") continue;
        if (await brk.wait()) continue;
        // finite budget exhausted: park until retryNow or a mode change
        this.dispatch({ type: "retry.exhausted" });
        if (this.closed) return;
        await this.notifier.wait(null);
        brk.reset();
      } else if (this.mode === "heartbeat") {
        await this.notifier.wait(this.heartbeatInterval);
        if (this.closed) return;
        if (this.mode === "heartbeat") await this.runProbe();
      } else await this.notifier.wait(null);
  }

  private warn({
    clockSkew,
    clockSkewExceeded,
    clientServerCompatible,
    clientVersion,
    nodeVersion,
  }: Status["details"]): void {
    if (clockSkewExceeded) {
      const direction = clockSkew.valueOf() > 0n ? "ahead of" : "behind";
      console.warn(
        `Measured excessive clock skew between this host and the Synnax Core. ` +
          `This host is ${direction} the Synnax Core by approximately ` +
          `${clockSkew.abs().toString()}.`,
      );
    }
    if (clientServerCompatible || this.versionWarned) return;
    this.versionWarned = true;
    const clientIsNewer =
      nodeVersion == null || migrate.semVerNewer(clientVersion, nodeVersion);
    const toUpgrade = clientIsNewer ? "Core" : "client";
    console.warn(
      `The Synnax Core version ${nodeVersion != null ? `${nodeVersion} ` : ""}is too ` +
        `${clientIsNewer ? "old" : "new"} for client version ${clientVersion}.\n` +
        `  This may cause compatibility issues. We recommend updating the ${toUpgrade}. For more information, see\n` +
        `  https://docs.synnaxlabs.com/reference/client/resources/troubleshooting#old-${toUpgrade.toLowerCase()}-version`,
    );
  }
}
