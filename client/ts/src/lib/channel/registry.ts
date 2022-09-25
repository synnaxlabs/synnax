import { ChannelPayload } from './ChannelPayload';
import ChannelRetriever from './ChannelRetriever';

export default class ChannelRegistry {
  private retriever: ChannelRetriever;
  private channels: Map<string, ChannelPayload>;

  constructor(retriever: ChannelRetriever) {
    this.retriever = retriever;
    this.channels = new Map();
  }

  async get(key: string): Promise<ChannelPayload> {
    let channel = this.channels.get(key);
    if (channel === undefined) {
      channel = (await this.retriever.retrieveByKeys(key))[0];
      this.channels.set(key, channel);
    }
    return channel;
  }

  async getN(...keys: string[]): Promise<ChannelPayload[]> {
    const results: ChannelPayload[] = [];
    const retrieveKeys: string[] = [];
    keys.forEach((key) => {
      const channel = this.channels.get(key);
      if (channel === undefined) retrieveKeys.push(key);
      else results.push(channel);
    });
    if (retrieveKeys.length > 0) {
      const channels = await this.retriever.retrieveByKeys(...retrieveKeys);
      channels.forEach((channel) => {
        this.channels.set(channel.key as string, channel);
        results.push(channel);
      });
    }
    return results;
  }
}
