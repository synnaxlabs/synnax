// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type CrudeTimeSpan, type destructor, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type theming } from "@/theming/aether";

const CONTEXT_KEY = "pluto-vis-staleness";

/// The sweep interval bounds how late a staleness transition can be reported. The
/// default sits well under the one second minimum staleness timeout.
export const DEFAULT_SWEEP_INTERVAL = TimeSpan.milliseconds(250);

/// stateZ carries the staleness fields a source-backed component adds to its own state.
/// Components that resolve the stale color on the worker add stalenessColor themselves;
/// DOM-rendered ones read it from their config instead.
export const stateZ = z.object({
  // stale reports that no sample has arrived within stalenessTimeout.
  stale: z.boolean().default(false),
  // Seconds without a sample before the source is considered stale.
  stalenessTimeout: z.number().default(5),
});

export interface EntryProps {
  /** Returns the current staleness timeout, in seconds. */
  timeout: () => number;
  /** Receives each staleness transition. */
  onChange: (stale: boolean) => void;
}

export interface Registration {
  /** Records an arriving sample. This clears staleness and restarts the countdown. */
  received: () => void;
  /** Releases the registration. */
  cleanup: destructor.Destructor;
}

interface Entry {
  props: EntryProps;
  lastReceived: number;
  stale: boolean;
}

/**
 * Tracker turns a registered source stale when no sample arrives within its timeout.
 *
 * One periodic sweep serves every registration, so the cost stays flat as sources and
 * sample rates grow. The sweep compares against the monotonic clock, so a throttled or
 * suspended worker resolves to the correct state when it wakes.
 */
class Tracker {
  private readonly entries = new Set<Entry>();
  private interval?: ReturnType<typeof setInterval>;
  private sweepInterval: TimeSpan;

  /** @param sweepInterval - How often to check registered sources. A bare number is
   * read as milliseconds. */
  constructor(sweepInterval: CrudeTimeSpan = DEFAULT_SWEEP_INTERVAL) {
    this.sweepInterval = TimeSpan.fromMilliseconds(sweepInterval);
  }

  /** @returns a registration for a source. Cleanup releases it. */
  register(props: EntryProps): Registration {
    // Registration counts as a sample, so a source that never sends turns stale one
    // window later instead of reading live forever.
    const entry: Entry = { props, lastReceived: performance.now(), stale: false };
    this.entries.add(entry);
    this.start();
    return {
      received: () => {
        entry.lastReceived = performance.now();
        this.set(entry, false);
      },
      cleanup: () => this.release(entry),
    };
  }

  /** Changes how often the tracker checks registered sources. Takes effect on the next
   * sweep. */
  setSweepInterval(next: CrudeTimeSpan): void {
    const span = TimeSpan.fromMilliseconds(next);
    if (span.equals(this.sweepInterval)) return;
    this.sweepInterval = span;
    if (this.interval == null) return;
    this.stop();
    this.start();
  }

  private start(): void {
    // Floor the delay: a sub-millisecond interval would peg the worker.
    const delay = Math.max(1, this.sweepInterval.milliseconds);
    this.interval ??= setInterval(() => this.sweep(), delay);
  }

  private stop(): void {
    clearInterval(this.interval);
    this.interval = undefined;
  }

  private release(entry: Entry): void {
    this.entries.delete(entry);
    if (this.entries.size === 0) this.stop();
  }

  private sweep(): void {
    const now = performance.now();
    this.entries.forEach((e) =>
      this.set(e, now - e.lastReceived >= e.props.timeout() * 1000),
    );
  }

  private set(entry: Entry, stale: boolean): void {
    if (stale === entry.stale) return;
    entry.stale = stale;
    entry.props.onChange(stale);
  }
}

// TimeSpan.z reads a bare number as nanoseconds. Take the number branch first so a
// plain number means milliseconds here, matching how the rest of pluto accepts a
// CrudeTimeSpan.
const sweepIntervalZ = z.union([
  z.number().transform((ms) => TimeSpan.milliseconds(ms)),
  TimeSpan.z,
]);

const providerStateZ = z.object({
  sweepInterval: sweepIntervalZ.default(DEFAULT_SWEEP_INTERVAL),
});

interface InternalState {
  tracker: Tracker;
}

export class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
  static readonly TYPE = "staleness.Provider";
  static readonly z = providerStateZ;
  schema = Provider.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const { sweepInterval } = this.state;
    i.tracker ??= new Tracker(sweepInterval);
    i.tracker.setSweepInterval(sweepInterval);
    ctx.set(CONTEXT_KEY, i.tracker, false);
  }
}

const use = (ctx: aether.Context): Tracker => ctx.get<Tracker>(CONTEXT_KEY);

/**
 * Registers a source with the tree's tracker, reusing `prev` when it is already
 * registered.
 * @param prev - The registration returned by an earlier call, if any.
 * @throws {NotFoundError} if no {@link Provider} is mounted above the caller.
 */
export const useRegistration = (
  ctx: aether.Context,
  prev: Registration | undefined,
  props: EntryProps,
): Registration => prev ?? use(ctx).register(props);

/**
 * Resolves the color that stale content renders in.
 * @param c - The configured staleness color. An unset (ZERO) color resolves to the
 * theme's warning shade.
 */
export const resolveColor = (
  c: color.Crude | undefined,
  theme: theming.Theme,
): color.Color =>
  c == null || color.isZero(c) ? theme.colors.warning.m1 : color.construct(c);

export const REGISTRY: aether.ComponentRegistry = { [Provider.TYPE]: Provider };
