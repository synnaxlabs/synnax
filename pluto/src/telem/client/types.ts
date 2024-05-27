import { TimeRange, type Series, type channel } from "@synnaxlabs/client";
import { type SeriesDigest } from "@synnaxlabs/x/telem";

export class ReadResponse {
  channel: channel.Payload;
  data: Series[];

  constructor(channel: channel.Payload, data: Series[]) {
    this.channel = channel;
    this.data = data;
  }

  get timeRange(): TimeRange {
    if (this.data.length === 0) return TimeRange.ZERO;
    const first = this.data[0].timeRange;
    const last = this.data[this.data.length - 1].timeRange;
    return new TimeRange(first.start, last.end);
  }

  get digest(): ReadResponseDigest {
    return {
      channel: this.channel.key,
      timeRange: this.timeRange.toPrettyString(),
      series: this.data.map((s) => s.digest),
    };
  }
}

export interface PromiseFns<T> {
  resolve: (value: T) => void;
  reject: (reason?: any) => void;
}

export interface ReadResponseDigest {
  channel: channel.Key;
  timeRange: string;
  series: SeriesDigest[];
}

export const responseDigests = (responses: ReadResponse[]): ReadResponseDigest[] =>
  responses.map((r) => r.digest);
