// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, Unreachable } from "@synnaxlabs/freighter";
import {
  breaker,
  ClockSkewCalculator,
  type CrudeTimeSpan,
  type destructor,
  errors,
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
/** Why the machine is in the error variant. */
export type Reason = z.infer<typeof reasonZ>;

export const stateZ = z.object({
  variant: status.variantZ,
  reason: reasonZ.optional(),
  message: z.string(),
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
/**
 * The connection state: a standard status variant projected over the fact
 * vector beneath it. `reason` is present iff `variant` is "error".
 */
export interface State extends z.infer<typeof stateZ> {}

export const DEFAULT_STATE: State = {
  variant: "disabled",
  message: "Disconnected",
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

const CHECK_ENDPOINT = "/connectivity/check";
const checkRequestZ = z.void();
const checkResponseZ = z.object({
  clusterKey: z.string(),
  nodeVersion: z.string().optional(),
  nodeTime: TimeStamp.z,
});

const createVersionWarning = (
  nodeVersion: string | null,
  clientVersion: string,
  clientIsNewer: boolean,
): string => {
  const toUpgrade = clientIsNewer ? "Core" : "client";
  return `The Synnax Core version ${nodeVersion != null ? `${nodeVersion} ` : ""}is too ${clientIsNewer ? "old" : "new"} for client version ${clientVersion}.
  This may cause compatibility issues. We recommend updating the ${toUpgrade}. For more information, see
  https://docs.synnaxlabs.com/reference/client/resources/troubleshooting#old-${toUpgrade.toLowerCase()}-version`;
};

export interface MachineParams {
  /** Probe transport. Must carry the full middleware chain, auth included. */
  unary: UnaryClient;
  clientVersion: string;
  /** Human-readable cluster name for state messages. */
  name?: string;
  /** Heartbeat cadence while connected. Defaults to 30 seconds. */
  heartbeatInterval?: CrudeTimeSpan;
  clockSkewThreshold?: CrudeTimeSpan;
  /**
   * Degraded-probe policy. `maxRetries` defaults to Infinity (never give up);
   * a finite value parks the machine in error(unreachable) once exhausted,
   * until {@link Machine.retryNow}. Escalation from warning/loading to
   * error(unreachable) happens after {@link MachineParams.escalateAfter}
   * attempts regardless.
   */
  retry?: breaker.Config;
  /** Failed attempts before escalating to error(unreachable). Defaults to 4. */
  escalateAfter?: number;
  /**
   * Brings the change stream up. Absent for detached (cache: false) clients,
   * in which case success does not require a live stream.
   */
  bringUpStream?: () => Promise<void>;
  /** Receives errors with no caller to throw to. */
  onInternalError?: (error: Error) => void;
}

const DEFAULT_RETRY: breaker.Config = {
  baseInterval: TimeSpan.seconds(1),
  maxInterval: TimeSpan.seconds(30),
  maxRetries: Infinity,
  scale: 2,
  jitter: 0.25,
};

/**
 * The client-owned connection state machine: one observable lifecycle over the
 * heartbeat probe, change-stream health, and auth outcomes. Constructed and
 * started by the Synnax client; consumers read {@link state} and subscribe via
 * {@link onChange}.
 */
export class Machine {
  private readonly params: Required<
    Pick<MachineParams, "clientVersion" | "escalateAfter">
  > &
    MachineParams;
  private readonly unary: UnaryClient;
  private readonly heartbeatInterval: TimeSpan;
  private readonly clockSkewThreshold: TimeSpan;
  private readonly retryConfig: breaker.Config;
  private readonly skewCalc = new ClockSkewCalculator();
  private readonly observer = new observe.Observer<State>();
  private live: State;
  private closed = false;
  private started = false;
  private versionWarned = false;
  private checking = false;
  private attempts = 0;
  private generation = 0;
  private wake: (() => void) | null = null;
  private wakePending = false;
  private loop: Promise<void> | null = null;
  private brk: breaker.Breaker | null = null;

  constructor(params: MachineParams) {
    this.params = { escalateAfter: 4, ...params };
    this.unary = params.unary;
    this.heartbeatInterval = new TimeSpan(
      params.heartbeatInterval ?? TimeSpan.seconds(30),
    );
    this.clockSkewThreshold = new TimeSpan(
      params.clockSkewThreshold ?? TimeSpan.seconds(1),
    ).abs();
    this.retryConfig = { ...DEFAULT_RETRY, ...params.retry };
    this.live = {
      ...DEFAULT_STATE,
      clientVersion: params.clientVersion,
      variant: "loading",
      message: `Connecting to ${params.name ?? "cluster"}...`,
    };
  }

  /** @returns a copy of the current connection state. */
  get state(): State {
    return {
      ...this.live,
      retry: this.live.retry == null ? null : { ...this.live.retry },
    };
  }

  /**
   * Subscribes to state changes. Fires on every transition and material fact
   * change. Returns a destructor that unsubscribes.
   */
  onChange(callback: (state: State) => void): destructor.Destructor {
    return this.observer.onChange(callback);
  }

  /** Starts the machine. Called once by the client after construction. */
  start(): void {
    if (this.started || this.closed) return;
    this.started = true;
    this.loop = this.run().catch((err: unknown) =>
      this.params.onInternalError?.(
        new Error("connection machine loop failed", { cause: err }),
      ),
    );
  }

  /** Resets the retry backoff and probes immediately. */
  retryNow(): void {
    if (this.closed) return;
    this.generation += 1;
    this.attempts = 0;
    this.brk?.reset();
    if (this.live.variant === "error" && this.live.reason === "unreachable")
      this.update({ variant: "loading", reason: undefined, message: "Connecting..." });
    this.wakeUp();
  }

  /**
   * Supplies new credentials after error(auth) and resumes connecting.
   * @param setCredentials - applies the credentials to the auth client.
   */
  reauthenticate(setCredentials: () => void): void {
    if (this.closed) return;
    setCredentials();
    this.generation += 1;
    this.attempts = 0;
    this.brk?.reset();
    this.update({
      variant: "loading",
      reason: undefined,
      error: undefined,
      message: "Connecting...",
    });
    this.wakeUp();
  }

  /** The change-stream feeder: the stream is live (first open or reopen). */
  notifyStreamLive(): void {
    if (this.closed) return;
    this.attempts = 0;
    this.update({
      streamLive: true,
      ...(this.live.variant !== "error" || this.live.reason === "unreachable"
        ? {
            variant: "success",
            reason: undefined,
            error: undefined,
            retry: null,
            message: `Connected to ${this.params.name ?? "cluster"}`,
          }
        : {}),
    });
  }

  /** The change-stream feeder: the stream dropped and is reconnecting. */
  notifyStreamDrop(error?: Error): void {
    if (this.closed || this.live.variant === "disabled") return;
    const wasHealthy = this.live.variant === "success";
    this.update({
      streamLive: false,
      ...(wasHealthy ? { variant: "warning", error, message: "Reconnecting..." } : {}),
    });
    if (wasHealthy) this.wake?.();
  }

  /** The cache feeder: mirrors the connection epoch into the state. */
  ingestEpoch(epoch: number): void {
    if (this.closed) return;
    this.update({ epoch });
  }

  /** The auth feeder: a definitive credential rejection. */
  notifyAuthFailure(error: Error): void {
    if (this.closed) return;
    this.update({
      variant: "error",
      reason: "auth",
      error,
      authenticated: false,
      retry: null,
      message: error.message,
    });
  }

  /** The auth feeder: a successful login. */
  notifyAuthSuccess(): void {
    if (this.closed) return;
    this.update({ authenticated: true });
  }

  /**
   * Resolves when the state satisfies the predicate; rejects on timeout.
   * The current state is evaluated first.
   */
  async waitFor(
    predicate: (state: State) => boolean,
    timeout?: CrudeTimeSpan,
  ): Promise<State> {
    if (predicate(this.live)) return this.state;
    return await new Promise<State>((resolve, reject) => {
      let timer: ReturnType<typeof setTimeout> | undefined;
      const detach = this.observer.onChange((state) => {
        if (!predicate(state)) return;
        if (timer != null) clearTimeout(timer);
        detach();
        resolve(state);
      });
      if (timeout != null)
        timer = setTimeout(() => {
          detach();
          reject(
            new Unreachable({
              message: `timed out waiting for connection after ${new TimeSpan(timeout).toString()}`,
            }),
          );
        }, new TimeSpan(timeout).milliseconds);
    });
  }

  /**
   * Resolves once the machine reaches success. Rejects with the typed failure
   * when it enters an error variant (the stored error: AuthError on auth
   * rejection, the transport error on unreachable escalation), on close, or on
   * timeout. The current state is evaluated first.
   */
  async awaitConnected(timeout?: CrudeTimeSpan): Promise<State> {
    const settle = (
      state: State,
      resolve: (s: State) => void,
      reject: (e: Error) => void,
    ): boolean => {
      if (state.variant === "success") {
        resolve(state);
        return true;
      }
      if (state.variant === "error" || state.variant === "disabled") {
        reject(state.error ?? new DisconnectedError(state.message));
        return true;
      }
      return false;
    };
    return await new Promise<State>((resolve, reject) => {
      let timer: ReturnType<typeof setTimeout> | undefined;
      const finish = (fn: () => void): void => {
        if (timer != null) clearTimeout(timer);
        detach();
        fn();
      };
      const detach = this.observer.onChange((state) => {
        settle(
          state,
          (s) => finish(() => resolve(s)),
          (e) => finish(() => reject(e)),
        );
      });
      if (
        settle(
          this.live,
          (s) => finish(() => resolve(s)),
          (e) => finish(() => reject(e)),
        )
      )
        return;
      if (timeout != null)
        timer = setTimeout(() => {
          detach();
          reject(
            new Unreachable({
              message: `timed out waiting for connection after ${new TimeSpan(timeout).toString()}`,
            }),
          );
        }, new TimeSpan(timeout).milliseconds);
    });
  }

  /** Stops the machine. Terminal; the state becomes disabled. */
  async stop(): Promise<void> {
    if (this.closed) return;
    this.closed = true;
    this.update({
      variant: "disabled",
      reason: undefined,
      streamLive: false,
      retry: null,
      message: "Closed",
    });
    this.wakeUp();
    await this.loop;
  }

  /**
   * Executes a single connectivity check and folds the result into the state.
   * Also used for one-shot probes via {@link checkConnection}.
   */
  async check(): Promise<State> {
    const measureSkew = !this.checking;
    // outcomes of probes issued before a retryNow/reauthenticate are stale and
    // must not overwrite the reset state
    const generation = this.generation;
    this.checking = true;
    try {
      if (measureSkew) this.skewCalc.start();
      const res = await this.unary.send(
        CHECK_ENDPOINT,
        undefined,
        checkRequestZ,
        checkResponseZ,
      );
      if (generation !== this.generation) return this.state;
      const facts: Partial<State> = {
        clusterKey: res.clusterKey,
        nodeVersion: res.nodeVersion,
        // the probe traverses the auth middleware, so a response is proof of auth
        authenticated: true,
      };
      if (measureSkew) {
        this.skewCalc.end(res.nodeTime);
        facts.clockSkew = this.skewCalc.skew;
        facts.clockSkewExceeded = this.skewCalc.exceeds(this.clockSkewThreshold);
        if (facts.clockSkewExceeded === true) {
          const direction = this.skewCalc.skew.valueOf() > 0n ? "ahead of" : "behind";
          console.warn(
            `Measured excessive clock skew between this host and ` +
              `the Synnax Core. This host is ${direction} the Synnax Core ` +
              `by approximately ${this.skewCalc.skew.abs().toString()}.`,
          );
        }
      }
      facts.clientServerCompatible = this.checkVersion(res.nodeVersion);
      this.update(facts);
      this.handleProbeSuccess();
    } catch (err) {
      if (generation === this.generation)
        this.handleProbeFailure(errors.fromUnknown(err));
    } finally {
      this.checking = false;
    }
    return this.state;
  }

  private checkVersion(nodeVersion: string | undefined): boolean {
    const { clientVersion } = this.params;
    if (nodeVersion == null) {
      if (!this.versionWarned) {
        console.warn(createVersionWarning(null, clientVersion, true));
        this.versionWarned = true;
      }
      return false;
    }
    const compatible = migrate.versionsEqual(clientVersion, nodeVersion, {
      checkMajor: true,
      checkMinor: true,
      checkPatch: false,
    });
    if (!compatible && !this.versionWarned) {
      console.warn(
        createVersionWarning(
          nodeVersion,
          clientVersion,
          migrate.semVerNewer(clientVersion, nodeVersion),
        ),
      );
      this.versionWarned = true;
    }
    return compatible;
  }

  private handleProbeSuccess(): void {
    this.attempts = 0;
    const { bringUpStream } = this.params;
    if (bringUpStream == null || this.live.streamLive) {
      if (this.live.variant !== "success" && this.live.variant !== "disabled")
        this.update({
          variant: "success",
          reason: undefined,
          error: undefined,
          retry: null,
          message: `Connected to ${this.params.name ?? "cluster"}`,
        });
      else this.update({ retry: null });
      return;
    }
    // reachable but the stream is down: stay degraded until it comes up, and
    // re-demand it in case its own retry budget was exhausted
    bringUpStream().catch((err: unknown) =>
      this.params.onInternalError?.(
        new Error("failed to bring up change stream", { cause: err }),
      ),
    );
  }

  private handleProbeFailure(error: Error): void {
    if (this.closed) return;
    if (AuthError.matches(error)) return this.notifyAuthFailure(error);
    this.attempts += 1;
    const escalate =
      this.attempts >= this.params.escalateAfter &&
      (this.live.variant === "loading" || this.live.variant === "warning");
    if (escalate)
      this.update({
        variant: "error",
        reason: "unreachable",
        error,
        message: error.message ?? "Cannot reach cluster",
      });
    else if (this.live.variant === "success")
      this.update({ variant: "warning", error, message: "Reconnecting..." });
    else this.update({ error });
  }

  private escalateExhausted(): void {
    if (this.live.variant !== "loading" && this.live.variant !== "warning") return;
    this.update({
      variant: "error",
      reason: "unreachable",
      message: this.live.error?.message ?? "Cannot reach cluster",
    });
  }

  private get probing(): boolean {
    if (this.closed) return false;
    const { variant, reason } = this.live;
    if (variant === "success") return false;
    if (variant === "disabled") return false;
    if (variant === "error") return reason === "unreachable";
    return true;
  }

  private async run(): Promise<void> {
    const brk = new breaker.Breaker({
      ...this.retryConfig,
      sleepFn: async (duration) => {
        this.update({
          retry: { attempt: this.attempts, nextAt: TimeStamp.now().add(duration) },
        });
        await this.sleep(duration);
      },
    });
    this.brk = brk;
    while (!this.closed) {
      if (this.probing) {
        await this.check();
        if (this.closed) return;
        if (this.probing) {
          if (!(await brk.wait())) {
            // finite retry budget exhausted: park until retryNow/reauthenticate
            this.escalateExhausted();
            this.update({ retry: null });
            await this.sleep(null);
            brk.reset();
          }
          continue;
        }
        brk.reset();
        this.update({ retry: null });
      }
      if (this.closed) return;
      if (this.live.variant === "success") {
        await this.sleep(this.heartbeatInterval);
        if (this.closed) return;
        if (this.live.variant === "success") await this.check();
      } else await this.sleep(null);
      brk.reset();
    }
  }

  /** Wakes the run loop: now if sleeping, else as soon as it next sleeps. */
  private wakeUp(): void {
    if (this.wake != null) this.wake();
    else this.wakePending = true;
  }

  /** Sleeps for the given span, or indefinitely when null, until woken. */
  private async sleep(span: TimeSpan | null): Promise<void> {
    if (this.wakePending) {
      this.wakePending = false;
      return;
    }
    await new Promise<void>((resolve) => {
      let timer: ReturnType<typeof setTimeout> | undefined;
      this.wake = () => {
        if (timer != null) clearTimeout(timer);
        this.wake = null;
        resolve();
      };
      if (span != null) timer = setTimeout(() => this.wake?.(), span.milliseconds);
    });
  }

  private update(partial: Partial<State>): void {
    const next: State = { ...this.live, ...partial };
    const changed =
      next.variant !== this.live.variant ||
      next.reason !== this.live.reason ||
      next.message !== this.live.message ||
      next.authenticated !== this.live.authenticated ||
      next.streamLive !== this.live.streamLive ||
      next.epoch !== this.live.epoch ||
      next.clusterKey !== this.live.clusterKey ||
      next.clientServerCompatible !== this.live.clientServerCompatible ||
      next.clockSkewExceeded !== this.live.clockSkewExceeded ||
      next.retry?.attempt !== this.live.retry?.attempt;
    this.live = next;
    if (changed) this.observer.notify(this.state);
  }
}
