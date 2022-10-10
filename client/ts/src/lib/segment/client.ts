import Registry from '../channel/registry';
import { TimeRange, TypedArray, UnparsedTimeStamp } from '../telem';
import Transport from '../transport';

import { TypedIterator } from './iterator';
import TypedSegment from './typed';
import { TypedWriter } from './writer';

export default class SegmentClient {
  private transport: Transport;
  private channels: Registry;

  constructor(transport: Transport, channels: Registry) {
    this.transport = transport;
    this.channels = channels;
  }

  /**
   * Opens a new iterator over the given channels within the provided time range.
   *
   * @param tr - A time range to iterate over.
   * @param keys - A list of channel keys to iterate over.
   * @param aggregate - Whether to accumulate iteration results or reset them
   * on every iterator method call.
   * @returns a new {@link TypedIterator}.
   */
  async newIterator(
    tr: TimeRange,
    keys: string[],
    aggregate: boolean
  ): Promise<TypedIterator> {
    const iter = new TypedIterator(
      this.transport.streamClient,
      this.channels,
      aggregate
    );
    await iter.open(tr, keys);
    return iter;
  }

  /**
   * Opens a new writer on the given channels.
   *
   * @param keys - The keys of the channels to write to. A writer cannot write to
   * a channel that is not in this list. See the {@link TypedWriter} documentation
   * for more information.
   * @returns a new {@link TypedWriter}.
   */
  async newWriter(keys: string[]): Promise<TypedWriter> {
    const writer = new TypedWriter(this.transport.streamClient, this.channels);
    await writer.open(keys);
    return writer;
  }

  /**
   * Writes telemetry to the given channel starting at the given timestamp.
   *
   * @param to - The key of the channel to write to.
   * @param start - The starting timestamp of the first sample in data.
   * @param data  - The telemetry to write. This telemetry must have the same
   * data type as the channel.
   * @throws if the channel does not exist.
   */
  async write(
    to: string,
    start: UnparsedTimeStamp,
    data: TypedArray
  ): Promise<boolean> {
    const writer = await this.newWriter([to]);
    try {
      return await writer.write(to, start, data);
    } finally {
      await writer.close();
    }
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param from - The key of the channel to read from.
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns a typed array containing the retrieved telemetry.
   * @throws if the channel does not exist.
   * @throws if the telemetry between start and end is not contiguous.
   */
  async read(
    from: string,
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<TypedArray> {
    return (await this.readSegment(from, start, end)).view;
  }

  /**
   * Reads a segment from the channel between the two timestamps.
   *
   * @param from - The key of the channel to read from.
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns a segment containing the retrieved telemetry.
   * @throws if the channel does not exist.
   * @throws if the telemetry between start and end is not contiguous.
   */
  async readSegment(
    from: string,
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<TypedSegment> {
    const iter = await this.newIterator(
      new TimeRange(start, end),
      [from],
      true
    );
    let seg: TypedSegment;
    try {
      await iter.first();
      // eslint-disable-next-line no-empty
      while (await iter.next()) {}
      seg = (await iter.value())[from];
    } finally {
      await iter.close();
    }
    return seg as TypedSegment;
  }
}
