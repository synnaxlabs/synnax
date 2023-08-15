import {
  Channel,
  ChannelKey,
  ChannelKeys,
  CrudeFrame,
  Series,
  Synnax,
  TimeStamp,
  Writer,
  Frame,
} from "@synnaxlabs/client";
import { z } from "zod";

import { TelemFactory } from "../factory";

import { AetherClient } from "@/client/aether";
import { AetherComposite } from "@/core/aether/worker";
import {
  NumericTelemSink,
  Telem,
  TelemContext,
  TelemProvider,
  TelemSpec,
  UseTelemResult,
} from "@/core/vis/telem";
import { TelemMeta } from "@/telem/base";

export const controllerState = z.object({
  authority: z.number(),
});

interface InternalState {
  client: Synnax | null;
  parent: TelemContext;
}

interface AetherControllerTelem {
  channelKeys: () => Promise<ChannelKeys>;
}

export class AetherController
  extends AetherComposite<typeof controllerState, InternalState>
  implements TelemProvider, TelemFactory
{
  registry: Map<AetherControllerTelem, null> = new Map();
  writer?: Writer;

  static readonly TYPE = "Controller";
  static readonly stateZ = controllerState;
  schema = AetherController.stateZ;

  afterUpdate(): void {
    console.log(this.ctx);
    this.internal.client = AetherClient.use(this.ctx);
    this.internal.parent = TelemContext.get(this.ctx);
    TelemContext.set(this.ctx, this);
  }

  private async channelKeys(): Promise<ChannelKeys> {
    const keys = new Set<ChannelKey>([]);
    for (const telem of this.registry.keys()) {
      const telemKeys = await telem.channelKeys();
      for (const key of telemKeys) keys.add(key);
    }
    return Array.from(keys);
  }

  async acquire(): Promise<void> {
    if (this.internal.client == null) return;

    this.writer = await this.internal.client.telem.newWriter(
      TimeStamp.now(),
      await this.channelKeys()
    );
  }

  async release(): Promise<void> {
    await this.writer?.close();
  }

  async set(frame: CrudeFrame): Promise<void> {
    console.log(frame);
    if (this.writer == null) await this.acquire();
    console.log(this.writer == null);
    const ack = await this.writer?.write(frame);
    if (ack == false) {
      console.log(await this.writer?.error());
    }
  }

  create(key: string, spec: TelemSpec, _: TelemFactory): Telem | null {
    if (spec.type === ControlledNumericTelemSink.TYPE) {
      const sink = new ControlledNumericTelemSink(key, this);
      this.registry.set(sink, null);
      return sink;
    }
    return null;
  }

  use<T>(key: string, spec: TelemSpec): UseTelemResult<T> {
    if (spec.type === ControlledNumericTelemSink.TYPE) {
      const sink = new ControlledNumericTelemSink(key, this);
      this.registry.set(sink, null);
      return [
        sink as unknown as T,
        () => {
          sink.cleanup();
          this.registry.delete(sink);
        },
      ];
    }
    return this.internal.parent.prov.use<T>(key, spec, this);
  }
}

export const controlNumericTelemSinkProps = z.object({
  channel: z.number(),
});

export type ControlNumericTelemSinkProps = z.infer<typeof controlNumericTelemSinkProps>;

export class ControlledNumericTelemSink
  extends TelemMeta<typeof controlNumericTelemSinkProps>
  implements NumericTelemSink
{
  controller: AetherController;
  static readonly TYPE = "controlled-numeric-telem-sink";
  channels: Channel[] = [];
  schema = controlNumericTelemSinkProps;

  constructor(key: string, controller: AetherController) {
    super(key);
    this.controller = controller;
  }

  async channelKeys(): Promise<ChannelKeys> {
    if (this.controller.internal.client == null) return [];
    const chan = await this.controller.internal.client?.channels.retrieve(
      this.props.channel
    );
    const keys = [chan.key];
    this.channels = [chan];
    console.log(this.channels);
    if (chan.index !== 0) {
      keys.push(chan.index);
      this.channels.push(
        await this.controller.internal.client?.channels.retrieve(chan.index)
      );
    }
    return keys;
  }

  invalidate(): void {}

  async set(value: number): Promise<void> {
    if (this.controller.internal.client == null) return;
    const ch = await this.controller.internal.client.channels.retrieve(
      this.props.channel
    );
    const ch2 = await this.controller.internal.client.channels.retrieve(ch.index);
    console.log(ch.key, ch2.key);
    const frame = new Frame(
      [ch.key, ch2.key],
      [
        new Series(new ch.dataType.Array([value])),
        new Series(new ch2.dataType.Array([BigInt(TimeStamp.now())])),
      ]
    );
    await this.controller.set(frame);
  }
}
