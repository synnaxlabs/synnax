// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ValidationError } from "@synnaxlabs/client";
import { type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { type Factory } from "@/telem/aether/factory";
import {
  AbstractSink,
  AbstractSource,
  type Sink,
  type SinkSpec,
  sinkSpecZ,
  type SinkTransformer,
  type Source,
  type SourceSpec,
  sourceSpecZ,
  type SourceTransformer,
  type Spec,
  type Telem,
} from "@/telem/aether/telem";

const connectionZ = z.object({
  from: z.string(),
  to: z.string(),
});

export const sourcePipelinePropsZ = z.object({
  connections: z.array(connectionZ),
  outlet: z.string(),
  segments: z.record(sourceSpecZ),
});

export type SourcePipelineProps = z.infer<typeof sourcePipelinePropsZ>;

export class PipelineFactory implements Factory {
  type = "pipeline";
  factory: Factory;

  constructor(factory: Factory) {
    this.factory = factory;
  }

  create(spec: Spec): Telem | null {
    switch (spec.type) {
      case SourcePipeline.TYPE:
        return new SourcePipeline(spec.props, this.factory);
      case SinkPipeline.TYPE:
        return new SinkPipeline(spec.props, this.factory);
      default:
        return null;
    }
  }
}

export class SourcePipeline<V>
  extends AbstractSource<typeof sourcePipelinePropsZ>
  implements Source<V>
{
  static readonly TYPE = "source-pipeline";
  schema = sourcePipelinePropsZ;
  sources: Record<string, Source<any> | SourceTransformer<any, any>> = {};

  private get outlet(): Source<V> {
    const { outlet } = this.props;
    const source = this.sources[outlet];
    if (source == null)
      throw new ValidationError(
        `[SourcePipeline] - expected source to exist at outlet '${outlet}', but none was found.`,
      );
    return source;
  }

  constructor(props: unknown, factory: Factory) {
    super(props);
    const { connections, segments } = this.props;
    Object.entries(segments).forEach(([id, spec]) => {
      const t = factory.create(spec);
      if (t == null) return;
      // Safe to do a cast here because we validated props with zod.
      this.sources[id] = t as Source<any>;
    });

    connections.forEach(({ from, to }) => {
      const fromSource = this.sources[from];
      const toSource = this.sources[to];
      if (fromSource == null || toSource == null) return;
      if ("setSources" in toSource) toSource.setSources({ [from]: fromSource });
    });
  }

  async value(): Promise<V> {
    return await this.outlet.value();
  }

  onChange(handler: () => void): Destructor {
    return this.outlet.onChange(handler);
  }

  async cleanup(): Promise<void> {
    await Promise.all(
      Object.values(this.sources).map(async (source) => await source.cleanup?.()),
    );
  }
}

export const sourcePipeline = <V extends string>(
  valueType: V,
  props: SourcePipelineProps,
): SourceSpec<V> => ({
  variant: "source",
  props,
  type: SourcePipeline.TYPE,
  valueType,
});

export const sinkPipelinePropsZ = z.object({
  connections: z.array(connectionZ),
  inlet: z.string(),
  segments: z.record(sinkSpecZ),
});

export type SinkPipelineProps = z.infer<typeof sinkPipelinePropsZ>;

export class SinkPipeline<V>
  extends AbstractSink<typeof sinkPipelinePropsZ>
  implements Sink<V>
{
  static readonly TYPE = "sink-pipeline";
  schema = sinkPipelinePropsZ;
  sinks: Record<string, Sink<any> | SinkTransformer<any, any>> = {};

  private get inlet(): Sink<V> {
    const { inlet } = this.props;
    const source = this.sinks[inlet];
    if (source == null)
      throw new ValidationError(
        `[SinkPipeline] - expected source to exist at inlet '${inlet}', but none was found.`,
      );
    return source;
  }

  constructor(props: unknown, factory: Factory) {
    super(props);
    const { connections, segments } = this.props;
    Object.entries(segments).forEach(([id, spec]) => {
      const t = factory.create(spec);
      if (t == null) return;
      // Safe to cast here because we validated props with zod.
      this.sinks[id] = t as Sink<any>;
    });

    connections.forEach(({ from, to }) => {
      const fromSink = this.sinks[from];
      const toSink = this.sinks[to];
      if (fromSink == null || toSink == null) return;
      if ("setSinks" in fromSink) fromSink.setSinks({ [to]: toSink });
    });
  }

  async set(value: V): Promise<void> {
    return await this.inlet.set(value);
  }

  async cleanup(): Promise<void> {
    await Promise.all(
      Object.values(this.sinks).map(async (sink) => await sink.cleanup?.()),
    );
  }
}

export const sinkPipeline = <V extends string>(
  valueType: V,
  props: SinkPipelineProps,
): SinkSpec<V> => ({
  variant: "sink",
  props,
  type: SinkPipeline.TYPE,
  valueType,
});
