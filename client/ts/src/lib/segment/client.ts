import Registry from '../channel/registry';
import { TimeRange, TypedArray, UnparsedTimeStamp } from '../telem';
import Transport from '../transport';

import { CoreIterator, SugaredIterator } from './iterator';
import Sugared from './sugared';

export default class SegmentClient {
  private transport: Transport;
  private channels: Registry;

  constructor(transport: Transport, channels: Registry) {
    this.transport = transport;
    this.channels = channels;
  }

  async newIterator(tr: TimeRange, keys: string[], aggregate: boolean) {
    const iter = new CoreIterator(this.transport.streamClient, aggregate);
    await iter.open(tr, keys);
    return iter;
  }

  async read(
    from: string,
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<TypedArray> {
    return (await this.readSegment(from, start, end)).view;
  }

  async readSegment(
    from: string,
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<Sugared> {
    const iter = new SugaredIterator(
      this.transport.streamClient,
      this.channels,
      true
    );
    let seg: Sugared;
    try {
      await iter.open(new TimeRange(start, end), [from]);
      await iter.first();
      // eslint-disable-next-line no-empty
      while (await iter.next()) {}
      seg = (await iter.value())[from];
    } finally {
      await iter.close();
    }
    return seg as Sugared;
  }
}
