// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { type diagram } from "@/vis/diagram/aether";

export const lightStateZ = z.object({
  enabled: z.boolean(),
  source: z.union([
    telem.booleanSourceSpecZ,
    z.object({
      channel: z.number(),
      threshold: z.object({
        lower: z.number().default(0.9),
        upper: z.number().default(1.1)
      }).default({ lower: 0.9, upper: 1.1 })
    })
  ]).optional().default(telem.noopBooleanSourceSpec),
});

export type LightState = z.input<typeof lightStateZ>;

interface InternalState {
  source: telem.BooleanSource;
  addStatus: status.Aggregate;
  stopListening: Destructor;
}

const createSourcePipeline = (source: LightState["source"]) => {
  if (!source) return telem.noopBooleanSourceSpec;
  if ("type" in source) return source;

  const threshold = source.threshold ?? { lower: 0.9, upper: 1.1 };
  
  return telem.sourcePipeline("boolean", {
    connections: [{ from: "valueStream", to: "threshold" }],
    segments: {
      valueStream: telem.streamChannelValue({ channel: source.channel }),
      threshold: telem.withinBounds({ 
        trueBound: { 
          lower: threshold.lower ?? 0.9, 
          upper: threshold.upper ?? 1.1
        } 
      }),
    },
    outlet: "threshold",
  });
};


// Light is a component that listens to a telemetry source to update its state.
export class Light
  extends aether.Leaf<typeof lightStateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "Light";

  schema = lightStateZ;

  async afterUpdate(): Promise<void> {
    this.internal.addStatus = status.useOptionalAggregate(this.ctx);
    const { source: sourceProps } = this.state;
    const { internal: i } = this;

    const source = createSourcePipeline(sourceProps);

    this.internal.source = await telem.useSource(
      this.ctx,
      source,
      this.internal.source,
    );

    await this.updateEnabledState();
    i.stopListening?.();
    i.stopListening = i.source.onChange(() =>
      this.updateEnabledState().catch(this.reportError.bind(this)),
    );
  }

  private reportError(e: Error): void {
    this.internal.addStatus({
      key: this.key,
      variant: "error",
      message: `Failed to update Light: ${e.message}`,
    });
  }

  private async updateEnabledState(): Promise<void> {
    const nextEnabled = await this.internal.source.value();
    if (nextEnabled !== this.state.enabled)
      this.setState((p) => ({ ...p, enabled: nextEnabled }));
  }

  async afterDelete(): Promise<void> {
    await this.internalAfterDelete();
  }

  private async internalAfterDelete(): Promise<void> {
    this.internal.stopListening();
    await this.internal.source.cleanup?.();
  }

  async render(): Promise<void> {}
}

export const REGISTRY: aether.ComponentRegistry = { [Light.TYPE]: Light };
