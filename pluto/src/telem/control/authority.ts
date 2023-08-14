import { ChannelKey, Synnax, TimeStamp, Writer } from "@synnaxlabs/client";
import { z } from "zod";

import { TelemMeta } from "../base";

import { AetherComposite } from "@/core/aether/worker";
import { NumericTelemSink, TelemProvider, UseTelemResult } from "@/core/vis/telem";

export const controllerState = z.object({
  authority: z.number(),
});

interface InternalState {
  client: Synnax;
}

export class Controller extends AetherComposite<typeof controllerState, InternalState> {
  registry: Map<ChannelKey, number> = new Map();
  writer?: Writer;

  afterUpdate(): void {
    this.internal.client = AetherCl;
  }

  register(key: ChannelKey): void {
    // increment key in registry or set to 1
    this.registry.set(key, (this.registry.get(key) ?? 0) + 1);
  }

  async acquire(): Promise<void> {
    this.writer = await this.client.telem.newWriter(
      TimeStamp.now(),
      Array.from(new Set(this.registry.values()))
    );
  }

  async release(): Promise<void> {
    await this.writer?.close();
  }

  async set(key: ChannelKey, value: number): Promise<void> {
    if (this.writer == null) await this.acquire();
    await this.writer?.write(key, new Float32Array([value]));
  }
}

export const controlNumericTelemSinkProps = z.object({
  channel: z.number(),
});

export class ControlledNumericTelemSink
  extends TelemMeta<typeof controlNumericTelemSinkProps>
  implements NumericTelemSink
{
  controller: Controller;

  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  invalidate(): void {}

  async set(value: number): Promise<void> {
    await this.controller.set(this.props.channel, value);
  }
}
