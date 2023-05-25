import { ReadResponse } from "@/telem/client";
import { ChannelRange } from "@/telem/range";
import { VisTelem } from "@/telem/telem";
import { AxisKey } from "@/vis/Axis";
import { VisContext } from "@/vis/context";
import { Channels } from "@/vis/line/channels";
import { Ranges } from "@/vis/line/ranges";

export class Telem {
  internal: VisTelem;
  private channels: Channels;

  constructor() {
    this.internal = new VisTelem();
    this.channels = new Channels();
  }

  private constructChannelRanges(channels: Channels, ranges: Ranges): ChannelRange[] {
    return ranges.array.map((r) => ({
      ...r,
      channels: channels.uniqueKeys,
    }));
  }

  async build(ctx: VisContext, channels: Channels, ranges: Ranges): Promise<void> {
    const cRanges = this.constructChannelRanges(channels, ranges);
    this.channels = channels;
    await this.internal.update(ctx.client, cRanges);
  }

  forEachAxis(f: (key: AxisKey, data: ReadResponse[]) => void): void {
    this.channels?.forEachAxis((channels, axis) => {
      const channelKeys = channels.map((c) => c.key);
      const data = this.internal.data
        .flat()
        .filter((d) => channelKeys.includes(d.channel.key));
      f(axis, data);
    });
  }

  axis(key: AxisKey): ReadResponse[] {
    const channels = this.channels.axis(key);
    const channelKeys = channels.map((c) => c.key);
    const data = this.internal.data
      .flat()
      .filter((d) => channelKeys.includes(d.channel.key));
    return data;
  }
}
