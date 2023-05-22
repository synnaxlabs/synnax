import { ChannelKey, ChannelName, ChannelParams } from "@/channel/payload";
import { ChannelRetriever, splitChannelParams } from "@/channel/retriever";
import { Frame } from "@/framer/frame";

export class BackwardFrameAdapter {
  private readonly adapter: Map<ChannelKey, ChannelName> | null;
  readonly keys: ChannelKey[];

  constructor(
    adapter: Map<ChannelKey, ChannelName> | null = null,
    keys: ChannelKey[] = []
  ) {
    this.adapter = adapter;
    this.keys = keys;
  }

  static async fromParams(
    retriever: ChannelRetriever,
    ...params: ChannelParams[]
  ): Promise<BackwardFrameAdapter> {
    const [keys, names] = splitChannelParams(params);
    if (names.length === 0) return new BackwardFrameAdapter(null, keys);
    const channels = await retriever.retrieve(...params);
    const adapter = new Map<ChannelKey, ChannelName>();
    names.forEach((name) => {
      const channel = channels.find((channel) => channel.name === name);
      if (channel == null) throw new Error(`Channel ${name} not found`);
      adapter.set(channel.key, channel.name);
    });
    return new BackwardFrameAdapter(
      adapter,
      channels.map((c) => c.key)
    );
  }

  adapt(fr: Frame): Frame {
    if (this.adapter == null) return fr;
    return fr.map((k, arr) => {
      if (typeof k === "number") {
        // @ts-expect-error
        const name = this.adapter.get(k);
        if (name == null) throw new Error(`Channel ${k} not found`);
        return [name, arr];
      }
      return [k, arr];
    });
  }
}

export class ForwardFrameAdapter {
  private readonly adapter: Map<ChannelName, ChannelKey> | null;
  readonly keys: ChannelKey[];

  constructor(
    adapter: Map<ChannelName, ChannelKey> | null = null,
    keys: ChannelKey[] = []
  ) {
    this.adapter = adapter;
    this.keys = keys;
  }

  static async fromParams(
    retriever: ChannelRetriever,
    ...params: ChannelParams[]
  ): Promise<ForwardFrameAdapter> {
    const [keys] = splitChannelParams(params);
    if (keys.length === 0) return new ForwardFrameAdapter();
    const channels = await retriever.retrieve(...params);
    const adapter = new Map<ChannelName, ChannelKey>();
    keys.forEach((key) => {
      const channel = channels.find((channel) => channel.key === key);
      if (channel == null) throw new Error(`Channel ${key} not found`);
      adapter.set(channel.name, channel.key);
    });
    return new ForwardFrameAdapter(
      adapter,
      channels.map((c) => c.key)
    );
  }

  adapt(fr: Frame): Frame {
    if (this.adapter == null) return fr;
    return fr.map((keyOrName, arr) => {
      if (typeof keyOrName === "string") {
        // @ts-expect-error
        const key = this.adapter.get(keyOrName);
        if (key == null) throw new Error(`Channel ${keyOrName} not found`);
        return [key, arr];
      }
      return [keyOrName, arr];
    });
  }
}
