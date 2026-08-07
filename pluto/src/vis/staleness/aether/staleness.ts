// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type destructor, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type theming } from "@/theming/aether";

const CONTEXT_KEY = "pluto-vis-staleness";

/// Seconds without a sample before a source is considered stale.
export const DEFAULT_TIMEOUT = 5;

/// configZ carries the staleness config every source-backed component adds to its own
/// state. Components that resolve the stale color on the worker add stalenessColor
/// themselves; DOM-rendered ones read it from their config instead.
export const configZ = z.object({
  stalenessTimeout: z.number().default(DEFAULT_TIMEOUT),
});

/// stateZ adds the reported staleness, which crosses to the DOM on every transition.
/// Extend it only when the DOM half renders the stale state. A component that draws on
/// the worker extends configZ and keeps staleness in its internal state.
export const stateZ = configZ.extend({
  // stale reports that no sample has arrived within stalenessTimeout.
  stale: z.boolean().default(false),
});

export interface EntryProps {
  /** Returns the current staleness timeout, in seconds. */
  timeout: () => number;
  /** Returns the staleness the source currently reports. */
  stale: () => boolean;
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
  // Milliseconds on the monotonic clock. Null until the first sample arrives.
  lastReceived: number | null;
}

// The source itself holds the last reported staleness, so a transition is a change
// against that instead of against a copy the Provider keeps.
const setStale = ({ props }: Entry, stale: boolean): void => {
  if (stale === props.stale()) return;
  props.onChange(stale);
};

// TimeSpan.z reads a bare number as nanoseconds. Take the number branch first so a
// plain number means milliseconds here, matching how the rest of pluto accepts a
// CrudeTimeSpan.
const sweepIntervalZ = z.union([
  z.number().transform((ms) => TimeSpan.milliseconds(ms)),
  TimeSpan.z,
]);

const providerStateZ = z.object({
  // Bounds how late a transition can be reported. Keep it well under the shortest
  // staleness timeout in use.
  sweepInterval: sweepIntervalZ,
});

/**
 * Provider turns a registered source stale when no sample arrives within its timeout.
 *
 * Staleness measures arrival, not sample time. It answers "is this source still
 * sending", not "is the newest sample recent". A source that delivers old data keeps
 * reading live, and a source that sends more slowly than its timeout reads stale even
 * while it is healthy. Give a source that sends on change a timeout longer than the
 * longest gap you expect between changes.
 *
 * One periodic sweep serves every source below the Provider, so the cost stays flat as
 * sources and sample rates grow. The sweep compares against the monotonic clock, so a
 * throttled or suspended worker resolves to the correct state when it wakes.
 */
export class Provider extends aether.Composite<typeof providerStateZ> {
  static readonly TYPE = "staleness.Provider";
  static readonly z = providerStateZ;
  schema = Provider.z;

  private readonly entries = new Set<Entry>();
  private interval?: NodeJS.Timeout;
  // The interval the running timer was started with. Null until the first sweep starts.
  private sweepInterval: TimeSpan | null = null;

  afterUpdate(ctx: aether.Context): void {
    this.updateSweepInterval();
    ctx.set(CONTEXT_KEY, this, false);
  }

  afterDelete(): void {
    this.entries.clear();
    this.stop();
  }

  /** @returns a registration for a source. Cleanup releases it. */
  register(props: EntryProps): Registration {
    const entry: Entry = { props, lastReceived: null };
    this.entries.add(entry);
    this.start();
    return {
      received: () => {
        entry.lastReceived = performance.now();
        setStale(entry, false);
      },
      cleanup: () => this.release(entry),
    };
  }

  // A new interval takes effect on the next sweep.
  private updateSweepInterval(): void {
    if (this.interval == null) return;
    if (this.sweepInterval?.equals(this.state.sweepInterval) === true) return;
    this.stop();
    this.start();
  }

  private start(): void {
    if (this.interval != null) return;
    this.sweepInterval = this.state.sweepInterval;
    // Floor the delay: a sub-millisecond interval would peg the worker.
    const delay = Math.max(1, this.sweepInterval.milliseconds);
    this.interval = setInterval(() => this.sweep(), delay);
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
    this.entries.forEach((e) => {
      // A source that has never sent reads as empty, not stale. Staleness means data
      // stopped, which cannot be true before any arrived.
      if (e.lastReceived == null) return;
      setStale(e, now - e.lastReceived >= e.props.timeout() * 1000);
    });
  }
}

const use = (ctx: aether.Context): Provider => ctx.get<Provider>(CONTEXT_KEY);

/**
 * Registers a source with the nearest {@link Provider}, reusing `prev` when it is
 * already registered.
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
