// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import type { destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import type { diagram } from "@/vis/diagram/aether";
import { staleness } from "@/vis/staleness/aether";

const readingZ = z.object({
  /** value is the newest value of the source, or null for no data and NaN. */
  value: z.number().nullable(),
  stale: z.boolean(),
});

/** Reading is what one source of a symbol node reports. */
export type Reading = z.infer<typeof readingZ>;

export const stateZ = z.object({
  stalenessTimeout: staleness.stateZ.shape.stalenessTimeout,
  sources: z.record(z.string(), telem.numberSourceSpecZ).default({}),
  sinks: z.record(z.string(), telem.numberSinkSpecZ).default({}),
  readings: z.record(z.string(), readingZ).default({}),
});

export const methodsZ = {
  set: z.function({ input: z.tuple([z.string(), z.number()]), output: z.void() }),
};

interface Reader {
  source: telem.NumberSource;
  staleness: staleness.Registration;
  stopListening: destructor.Destructor;
}

const NO_READING: Reading = { value: null, stale: false };

// Keys come from user data, so a key such as "constructor" must not find a prototype
// member.
const own = <V>(r: Record<string, V>, key: string): V | undefined =>
  Object.hasOwn(r, key) ? r[key] : undefined;

const read = (source: telem.NumberSource): number | null => {
  const { data } = source.value();
  return Number.isNaN(data) ? null : data;
};

const release = ({ source, staleness, stopListening }: Reader): void => {
  stopListening();
  staleness.cleanup();
  source.cleanup?.();
};

/**
 * SymbolNode reads the channels of one symbol node and writes its commands. It reports
 * a reading for each source by its key, and writes a value to a sink by its key.
 */
export class SymbolNode
  extends aether.Leaf<typeof stateZ, {}, typeof methodsZ>
  implements diagram.Element, aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "SymbolNode";
  static readonly METHODS = methodsZ;
  static readonly z = stateZ;

  override schema = stateZ;
  override methods = methodsZ;

  private readonly readers = new Map<string, Reader>();
  private readonly sinks = new Map<string, telem.NumberSink>();

  override afterUpdate(ctx: aether.Context): void {
    const { sources, sinks } = this.state;
    for (const [key, reader] of this.readers)
      if (!Object.hasOwn(sources, key)) {
        release(reader);
        this.readers.delete(key);
      }
    for (const [key, spec] of Object.entries(sources))
      this.updateReader(ctx, key, spec);
    for (const [key, sink] of this.sinks)
      if (!Object.hasOwn(sinks, key)) {
        sink.cleanup?.();
        this.sinks.delete(key);
      }
    for (const [key, spec] of Object.entries(sinks))
      this.sinks.set(key, telem.useSink(ctx, spec, this.sinks.get(key)));
    if (Object.keys(this.state.readings).some((key) => !Object.hasOwn(sources, key)))
      this.setState((p) => ({
        ...p,
        readings: Object.fromEntries(
          Object.entries(p.readings).filter(([key]) => Object.hasOwn(p.sources, key)),
        ),
      }));
  }

  private updateReader(ctx: aether.Context, key: string, spec: telem.Spec): void {
    const prev = this.readers.get(key);
    const source = telem.useSource<number>(ctx, spec, prev?.source);
    const registration = staleness.useRegistration(
      ctx,
      prev?.staleness,
      {
        timeout: () => this.state.stalenessTimeout,
        stale: () => own(this.state.readings, key)?.stale ?? false,
        onChange: (stale) => this.setReading(key, { stale }),
      },
      source,
    );
    prev?.stopListening();
    const stopListening = source.onChange(() => {
      registration.received();
      this.setReading(key, { value: read(source) });
    });
    this.readers.set(key, { source, staleness: registration, stopListening });
    this.setReading(key, { value: read(source) });
  }

  private setReading(key: string, change: Partial<Reading>): void {
    const prev = own(this.state.readings, key);
    const next = { ...(prev ?? NO_READING), ...change };
    if (prev?.value === next.value && prev.stale === next.stale) return;
    this.setState((p) => ({ ...p, readings: { ...p.readings, [key]: next } }));
  }

  /**
   * Writes value to the sink of key.
   * @throws {NotFoundError} if the state has no sink for key.
   */
  set(key: string, value: number): void {
    const sink = this.sinks.get(key);
    if (sink == null) throw new NotFoundError(`symbol node has no sink ${key}`);
    sink.set(value);
  }

  override afterDelete(): void {
    this.readers.forEach(release);
    this.readers.clear();
    this.sinks.forEach((sink) => sink.cleanup?.());
    this.sinks.clear();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [SymbolNode.TYPE]: SymbolNode };
