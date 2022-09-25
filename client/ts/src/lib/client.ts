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

/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 */
export default class Synnax {
  transport: Transport;
  data: SegmentClient;
  channel: ChannelClient;

  /**
   * @param host - Hostname of a node in the cluster.
   * @param port - Port of the node in the cluster.
   */
  constructor({ host, port }: ClientProps) {
    this.transport = new Transport(new URL({ host, port }));
    const chRetriever = new ChannelRetriever(this.transport);
    const chCreator = new ChannelCreator(this.transport);
    const chRegistry = new Registry(chRetriever);
    this.data = new SegmentClient(this.transport, chRegistry);
    this.channel = new ChannelClient(this.data, chRetriever, chCreator);
  }
}
