import { URL } from '@synnaxlabs/freighter';

import ChannelClient from './channel/client';
import ChannelCreator from './channel/creator';
import Registry from './channel/registry';
import ChannelRetriever from './channel/retriever';
import SegmentClient from './segment/client';
import Transport from './transport';

export type ClientProps = {
  host: string;
  port: number;
};

export default class Synnax {
  transport: Transport;
  data: SegmentClient;
  channel: ChannelClient;

  constructor({ host, port }: ClientProps) {
    this.transport = new Transport(new URL({ host, port }));
    const channelRetriever = new ChannelRetriever(this.transport);
    const channelCreator = new ChannelCreator(this.transport);
    const channelRegistry = new Registry(channelRetriever);
    this.data = new SegmentClient(this.transport, channelRegistry);
    this.channel = new ChannelClient(
      this.data,
      channelRetriever,
      channelCreator
    );
  }
}
