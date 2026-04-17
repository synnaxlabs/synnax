// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, type TimeRange } from "@synnaxlabs/client";
import { MultiSeries, type Series, Size } from "@synnaxlabs/x";

import { Dynamic, type DynamicProps } from "@/telem/client/cache/dynamic";
import {
  type CacheGCMetrics as CacheGCResult,
  type DirtyReadResult,
  Static,
  type StaticProps,
} from "@/telem/client/cache/static";

interface UnaryProps extends StaticProps, Pick<DynamicProps, "dynamicBufferSize"> {
  channel: channel.Payload;
}

/**
 * NATIVE_FIDELITY is the alignmentMultiple value representing a series at
 * native (unreduced) resolution. All live and raw historical data flows into
 * this tier; server-reduced data lives in tiers keyed by its alignmentMultiple.
 */
export const NATIVE_FIDELITY: bigint = 1n;

/**
 * Unary holds the cache state for a single channel: a Dynamic buffer for the
 * live tail (always at native fidelity) and one or more Static tiers keyed by
 * the alignmentMultiple of the series within them. Each tier enforces its own
 * non-overlapping alignment invariant, and the cross-tier layered read in
 * dirtyReadAtFidelity walks tiers from coarsest to finest so a coarse request
 * can be satisfied by any finer-or-equal cached data without refetching.
 */
export class Unary {
  readonly channel: channel.Payload;
  private closed: boolean = false;
  private readonly ins: alamos.Instrumentation;
  private readonly staticProps: StaticProps;
  private readonly statics: Map<bigint, Static>;
  private readonly dynamic: Dynamic;

  constructor(props: UnaryProps) {
    this.channel = props.channel;
    this.ins = props.instrumentation ?? alamos.NOOP;
    this.staticProps = props;
    this.statics = new Map();
    this.statics.set(NATIVE_FIDELITY, new Static(props));
    this.dynamic = new Dynamic({
      dynamicBufferSize: props.dynamicBufferSize,
      dataType: this.channel.dataType,
    });
  }

  writeDynamic(series: MultiSeries): MultiSeries {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted dynamic write to a closed cache for channel ${this.channel.name}`,
      );
      return new MultiSeries([]);
    }
    const { flushed, allocated } = this.dynamic.write(series);
    // Buffers flushed from the dynamic cache are native-rate live data and
    // always go into the native static tier.
    if (flushed.length > 0) this.nativeStatic().write(flushed);
    return allocated;
  }

  get leadingBuffer(): Series | null {
    return this.dynamic.leadingBuffer;
  }

  /**
   * Convenience wrapper around writeStaticAtFidelity for data known to be at
   * native fidelity.
   */
  writeStatic(series: MultiSeries): void {
    this.writeStaticAtFidelity(series, NATIVE_FIDELITY);
  }

  /**
   * Writes the given series into the Static tier keyed by the given
   * alignmentMultiple, creating the tier lazily if it does not yet exist.
   */
  writeStaticAtFidelity(series: MultiSeries, fidelity: bigint): void {
    if (this.closed)
      return this.ins.L.warn(
        `Ignoring attempted static write to a closed cache for channel ${this.channel.name}`,
      );
    const normalized = fidelity === 0n ? NATIVE_FIDELITY : fidelity;
    let tier = this.statics.get(normalized);
    if (tier == null) {
      tier = new Static(this.staticProps);
      this.statics.set(normalized, tier);
    }
    tier.write(series);
  }

  /**
   * Convenience wrapper around dirtyReadAtFidelity for callers that want
   * native-fidelity data only.
   */
  read(tr: TimeRange): DirtyReadResult {
    return this.dirtyReadAtFidelity(tr, NATIVE_FIDELITY);
  }

  /**
   * Satisfies a read for the given time range at the required fidelity,
   * where requiredFidelity is an upper bound on alignmentMultiple and lower
   * values are higher fidelity. The lookup walks Static tiers from coarsest
   * to finest, each time asking the tier to fill as much of the remaining gap
   * as it can. Any range not covered by any qualifying tier is returned as a
   * gap for the caller to fetch.
   */
  dirtyReadAtFidelity(tr: TimeRange, requiredFidelity: bigint): DirtyReadResult {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted dirty read from a closed cache for channel ${this.channel.name}`,
      );
      return { series: new MultiSeries([]), gaps: [tr] };
    }
    const normalized = requiredFidelity === 0n ? NATIVE_FIDELITY : requiredFidelity;
    // Qualifying tiers have alignmentMultiple <= requiredFidelity; walked
    // coarsest-first to prefer cheaper-to-render data when available.
    const qualifyingKeys = [...this.statics.keys()]
      .filter((k) => k <= normalized)
      .sort((a, b) => (a < b ? 1 : a > b ? -1 : 0));
    let remaining: TimeRange[] = [tr];
    const collected: Series[] = [];
    for (const fidelity of qualifyingKeys) {
      if (remaining.length === 0) break;
      const tier = this.statics.get(fidelity)!;
      const next: TimeRange[] = [];
      for (const sub of remaining) {
        const { series, gaps } = tier.dirtyRead(sub);
        collected.push(...series.series);
        next.push(...gaps);
      }
      remaining = next;
    }
    return { series: new MultiSeries(collected), gaps: remaining };
  }

  gc(): CacheGCResult {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted garbage collection on a closed cache for channel ${this.channel.name}`,
      );
      return { purgedSeries: 0, purgedBytes: Size.bytes(0) };
    }
    let purgedSeries = 0;
    let purgedBytes = Size.bytes(0);
    for (const tier of this.statics.values()) {
      const res = tier.gc();
      purgedSeries += res.purgedSeries;
      purgedBytes = purgedBytes.add(res.purgedBytes);
    }
    return { purgedSeries, purgedBytes };
  }

  close(): void {
    this.closed = true;
    this.dynamic.close();
    for (const tier of this.statics.values()) tier.close();
    this.statics.clear();
  }

  private nativeStatic(): Static {
    let tier = this.statics.get(NATIVE_FIDELITY);
    if (tier == null) {
      tier = new Static(this.staticProps);
      this.statics.set(NATIVE_FIDELITY, tier);
    }
    return tier;
  }
}
