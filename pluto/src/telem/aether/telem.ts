// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ValidationError } from "@synnaxlabs/client";
import { type bounds, type Destructor, observe, type Series } from "@synnaxlabs/x";
import { z } from "zod";

import { type color } from "@/color/core";
import { type status } from "@/status/aether";

const transferrable = z.instanceof(ArrayBuffer);

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export const specZ = z.object({
  type: z.string(),
  variant: z.enum(["source", "sink", "source-transformer", "sink-transformer"]),
  valueType: z.string(),
  props: z.any(),
  transfer: z.array(transferrable).optional(),
});

export const sourceSpecZ = specZ.extend({ variant: z.literal("source") });
export const sinkSpecZ = specZ.extend({ variant: z.literal("sink") });
export const sourceTransformerSpecZ = specZ.extend({
  variant: z.literal("source-transformer"),
});
export const sinkTransformerSpecZ = specZ.extend({
  variant: z.literal("sink-transformer"),
});

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export type Spec = z.infer<typeof specZ>;

export type SourceSpec<V extends string> = z.infer<typeof sourceSpecZ> & {
  valueType: V;
};

export type SinkSpec<V extends string> = z.infer<typeof sinkSpecZ> & {
  valueType: V;
};

export interface ValueProps {
  onLoad: () => void;
}

export interface Telem {
  cleanup?: () => Promise<void>;
}

export interface Source<V> extends Telem, observe.Observable<void> {
  value: (props?: ValueProps) => Promise<V>;
}

export interface Sink<V> extends Telem {
  set: (value: V) => Promise<void>;
}

export interface SourceTransformer<I, O> extends Telem, Source<O> {
  setSources: (sources: Record<string, Source<I>>) => void;
}

export interface SinkTransformer<I, O> extends Telem, Sink<I> {
  setSinks: (sinks: Record<string, Sink<O>>) => void;
}

export type SeriesSource = Source<[bounds.Bounds, Series[]]>;
export const seriesSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("series") });
export type SeriesSourceSpec = z.infer<typeof seriesSourceSpecZ>;

export type BooleanSource = Source<boolean>;
export const booleanSourceSpecZ = sourceSpecZ.extend({
  valueType: z.literal("boolean"),
});
export type BooleanSourceSpec = z.infer<typeof booleanSourceSpecZ>;

export type BooleanSink = Sink<boolean>;
export const booleanSinkSpecZ = sinkSpecZ.extend({ valueType: z.literal("boolean") });
export type BooleanSinkSpec = z.infer<typeof booleanSinkSpecZ>;

export type BooleanSinkTransformer = SinkTransformer<boolean, number>;
export const booleanSinkTransformerSpecZ = sinkTransformerSpecZ.extend({
  valueType: z.literal("boolean"),
});
export type BooleanSinkTransformerSpec = z.infer<typeof booleanSinkTransformerSpecZ>;

export type BooleanSourceTransformer = SourceTransformer<number, boolean>;
export const booleanSourceTransformerSpecZ = sourceTransformerSpecZ.extend({
  valueType: z.literal("boolean"),
});
export type BooleanSourceTransformerSpec = z.infer<
  typeof booleanSourceTransformerSpecZ
>;

export type NumberSource = Source<number>;
export const numberSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("number") });
export type NumberSourceSpec = z.infer<typeof numberSourceSpecZ>;

export type NumberSink = Sink<number>;
export const numberSinkSpecZ = sinkSpecZ.extend({ valueType: z.literal("number") });
export type NumberSinkSpec = z.infer<typeof numberSinkSpecZ>;

export type ColorSource = Source<color.Color>;
export const colorSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("color") });
export type ColorSourceSpec = z.infer<typeof colorSourceSpecZ>;

export type StatusSource = Source<status.Spec>;
export const statusSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("status") });
export type StatusSourceSpec = z.infer<typeof statusSourceSpecZ>;

export type StringSource = Source<string>;
export const stringSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("string") });
export type StringSourceSpec = z.infer<typeof stringSourceSpecZ>;

export class Base<P extends z.ZodTypeAny> extends observe.BaseObserver<void> {
  private props_: z.output<P> | undefined = undefined;
  private readonly uProps_: unknown | undefined = undefined;
  schema: P | undefined = undefined;

  constructor(props: unknown) {
    super();
    this.uProps_ = props;
  }

  get props(): z.output<P> {
    if (this.props_ == null) {
      const res = this._schema.safeParse(this.uProps_);
      if (res.success) this.props_ = res.data;
      else
        throw new ValidationError(
          `[BaseTelem] - expected props to be valid, but found the following errors:
          ${res.error.message}`,
        );
    }
    return this.props_;
  }

  private get _schema(): P {
    if (this.schema == null)
      throw new ValidationError(
        `[BaseTelem] - expected subclass to define props schema, but none was found.
    Make sure to define a property 'schema' on the class.`,
      );
    return this.schema;
  }

  async cleanup(): Promise<void> {}
}

export class AbstractSource<P extends z.ZodTypeAny> extends Base<P> {}

export class AbstractSink<P extends z.ZodTypeAny> extends Base<P> {}

export class UnarySourceTransformer<I, O, P extends z.ZodTypeAny>
  extends AbstractSource<P>
  implements SourceTransformer<I, O>
{
  sources: Record<string, Source<I>> = {};

  private get source(): Source<I> {
    const [source] = Object.values(this.sources);
    if (source == null)
      throw new ValidationError(
        `[UnarySourceTransformer] - expected source to exist, but none was found.`,
      );
    return source;
  }

  async value(): Promise<O> {
    return this.transform(await this.source.value());
  }

  onChange(handler: () => void): Destructor {
    return this.source.onChange(() => {
      this.source
        .value()
        .then((value) => {
          if (this.shouldNotify(value)) handler();
        })
        .catch(console.error);
    });
  }

  setSources(sources: Record<string, Source<I>>): void {
    this.sources = sources;
  }

  protected shouldNotify(_: I): boolean {
    return true;
  }

  protected transform(_: I): O {
    throw new ValidationError(
      `[UnarySourceTransformer] - expected subclass to define transform method, but none was found.
      Make sure to define a method 'transform' on the class.`,
    );
  }
}

export class MultiSourceTransformer<I, O, P extends z.ZodTypeAny>
  extends AbstractSource<P>
  implements SourceTransformer<I, O>
{
  sources: Record<string, Source<I>> = {};

  async value(): Promise<O> {
    const values = Object.fromEntries(
      await Promise.all(
        Object.entries(this.sources).map(async ([id, source]) => [
          id,
          await source.value(),
        ]),
      ),
    ) as Record<string, I>;
    return this.transform(values);
  }

  setSources(sources: Record<string, Source<I>>): void {
    this.sources = { ...this.sources, ...sources };
  }

  protected transform(_: Record<string, I>): O {
    throw new ValidationError(
      `[MultiSourceTransformer] - expected subclass to define transform method, but none was found.
      Make sure to define a method 'transform' on the class.`,
    );
  }
}

export class UnarySinkTransformer<I, O, P extends z.ZodTypeAny>
  extends Base<P>
  implements SinkTransformer<I, O>
{
  sinks: Record<string, Sink<O>> = {};

  private get sink(): Sink<O> {
    const [sink] = Object.values(this.sinks);
    if (sink == null)
      throw new ValidationError(
        `[UnarySinkTransformer] - expected sink to exist, but none was found.`,
      );
    return sink;
  }

  async set(value: I): Promise<void> {
    return await this.sink.set(this.transform(value));
  }

  setSinks(sinks: Record<string, Sink<O>>): void {
    this.sinks = { ...this.sinks, ...sinks };
  }

  protected transform(_: I): O {
    throw new ValidationError(
      `[UnarySinkTransformer] - expected subclass to define transform method, but none was found.
      Make sure to define a method 'transform' on the class.`,
    );
  }
}
