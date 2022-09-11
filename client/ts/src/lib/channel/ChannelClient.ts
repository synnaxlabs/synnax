import { UnaryClient } from '@arya-analytics/freighter';
import Channel from './Channel';
import Transport from '../transport';

type ChannelRetrieveRequest = {
  keys?: string[];
  node_id?: number;
  names?: string[];
};

type ChannelRetrieveResponse = {
  channels: Channel[];
};

type ChannelCreateRequest = {
  channel: Channel;
  count: number;
};

const retrieveEndpoint = '/channel/retrieve';
const createEndpoint = '/channel/create';

export default class ChannelClient {
  retrieve_transport: UnaryClient<
    ChannelRetrieveRequest,
    ChannelRetrieveResponse
  >;
  create_transport: UnaryClient<ChannelCreateRequest, Channel>;

  constructor(transport: Transport) {
    this.retrieve_transport = transport.http.get();
    this.create_transport = transport.http.post();
  }

  async retrieve(...keys: string[]): Promise<Channel[]> {
    return this._retrieve({ keys });
  }

  async retrieveByNodeID(node_id: number): Promise<Channel[]> {
    return this._retrieve({ node_id });
  }

  async retrieveByName(...names: string[]): Promise<Channel[]> {
    return this._retrieve({ names });
  }

  async create(channel: Channel): Promise<Channel> {
    return this.createN(channel, 1);
  }

  async createN(channel: Channel, count: number = 1): Promise<Channel> {
    const [res, err] = await this.create_transport.send(createEndpoint, {
      channel,
      count,
    });
    if (err) {
      throw err;
    }
    return res;
  }

  private async _retrieve(req: ChannelRetrieveRequest): Promise<Channel[]> {
    const [res, err] = await this.retrieve_transport.send(
      retrieveEndpoint,
      req
    );
    if (err) {
      throw err;
    }
    return res.channels;
  }
}
