// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  breaker,
  type CrudeTimeSpan,
  errors,
  sync,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type Client } from "@/connection/client";
import { type Event, type State } from "@/connection/state";

/** How hard the prober is working. */
export type Mode = "probing" | "heartbeat" | "idle";

/** The mode a given connection state calls for. */
export const modeFor = ({ variant, details }: State): Mode => {
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
  maxInterval: TimeSpan.seconds(30),
  maxRetries: Infinity,
  scale: 2,
  jitter: 0.25,
};

export interface ProberParams {
  client: Client;
  /**
   * Degraded-probe policy. `maxRetries` defaults to Infinity (never give up);
   * a finite value parks the prober once exhausted, until {@link
   * Prober.retryNow}.
   */
  retry?: breaker.Config;
  /** Probe cadence while connected. Defaults to 30 seconds. */
  heartbeatInterval?: CrudeTimeSpan;
  emit: (event: Event) => void;
  /** Receives errors with no caller to throw to. */
  onInternalError?: (error: Error) => void;
}

/**
 * Runs the connectivity probe on a cadence, backing off while degraded. Holds
 * no connection state: it emits probe and retry events and is told what mode to
 * run in.
 */
export class Prober {
  private readonly client: Client;
  private readonly heartbeatInterval: TimeSpan;
  private readonly retryConfig: breaker.Config;
  private readonly emit: (event: Event) => void;
  private readonly onInternalError?: (error: Error) => void;
  private readonly notifier = new sync.Notifier();
  private mode: Mode = "probing";
  private closed = false;
  private started = false;
  private attempts = 0;
  private generation = 0;
  private loop: Promise<void> | null = null;
  private brk: breaker.Breaker | null = null;

  constructor(params: ProberParams) {
    this.client = params.client;
    this.emit = params.emit;
    this.onInternalError = params.onInternalError;
    this.heartbeatInterval = new TimeSpan(
      params.heartbeatInterval ?? TimeSpan.seconds(30),
    );
    this.retryConfig = { ...DEFAULT_RETRY, ...params.retry };
  }

  /** Starts the probe loop. Called once by the client after construction. */
  start(): void {
    if (this.started || this.closed) return;
    this.started = true;
    this.loop = this.run().catch((err: unknown) =>
      this.onInternalError?.(new Error("connection prober failed", { cause: err })),
    );
  }

  /** Switches modes. Idempotent; resets the backoff on every real change. */
  setMode(mode: Mode): void {
    if (this.closed || this.mode === mode) return;
    if (mode !== "probing") this.attempts = 0;
    this.mode = mode;
    this.brk?.reset();
    this.notifier.notify();
  }

  /** Resets the backoff and probes immediately, discarding in-flight results. */
  retryNow(): void {
    if (this.closed) return;
    this.generation += 1;
    this.attempts = 0;
    this.brk?.reset();
    this.notifier.notify();
  }

  /** Stops the probe loop. Terminal. */
  async stop(): Promise<void> {
    if (this.closed) return;
    this.closed = true;
    this.notifier.notify();
    await this.loop;
  }

  private async probe(): Promise<void> {
    // results of probes issued before a retryNow are stale and must not reach
    // the reducer, which has already been reset
    const generation = this.generation;
    try {
      const info = await this.client.check();
      if (generation !== this.generation || this.closed) return;
      this.attempts = 0;
      this.emit({ type: "probe.success", info });
    } catch (err) {
      if (generation !== this.generation || this.closed) return;
      this.attempts += 1;
      this.emit({
        type: "probe.failure",
        error: errors.fromUnknown(err),
        attempt: this.attempts,
      });
    }
  }

  private async run(): Promise<void> {
    const brk = new breaker.Breaker({
      ...this.retryConfig,
      sleepFn: async (duration) => {
        this.emit({
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
        await this.probe();
        if (this.closed) return;
        if (this.mode !== "probing") continue;
        if (await brk.wait()) continue;
        // finite budget exhausted: park until retryNow or a mode change
        this.emit({ type: "retry.exhausted" });
        if (this.closed) return;
        await this.notifier.wait(null);
        brk.reset();
      } else if (this.mode === "heartbeat") {
        await this.notifier.wait(this.heartbeatInterval);
        if (this.closed) return;
        if (this.mode === "heartbeat") await this.probe();
      } else await this.notifier.wait(null);
  }
}
