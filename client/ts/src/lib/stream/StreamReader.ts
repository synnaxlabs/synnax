import { ClientStream } from '@arya-analytics/freighter';
import Transport from '../transport';
import { Sample } from './sample';

export type StreamClientProps = {
  transport: Transport;
};

const endpoint = '/stream/read';

type StreamReaderRequest = {
  channelKeys: string[];
};

type StreamReaderResponse = {
  samples: Sample[];
};

export default class StreamReader {
  transport: Transport;
  is_open: boolean;
  stream: ClientStream<StreamReaderRequest, StreamReaderResponse>;
  currentKeys: string[];

  constructor({ transport }: StreamClientProps) {
    this.transport = transport;
  }

  async maybe_open() {
    if (!this.is_open)
      this.stream = await this.transport.stream.stream<
        StreamReaderRequest,
        StreamReaderResponse
      >(endpoint);
  }

  async maybe_close() {
    if (this.is_open) this.stream.closeSend();
  }

  async subscribe(...channelKeys: string[]) {
    this.currentKeys.push(...channelKeys);
    if (!this.is_open) {
      await this.maybe_open();
    }
    this.stream.send({ channelKeys: this.currentKeys });
  }

  async unsubscribe(...channelKeys: string[]) {
    if (!this.is_open) {
      throw new Error('Cannot unsubscribe from a closed stream');
    }
    this.currentKeys = this.currentKeys.filter(
      (key) => !channelKeys.includes(key)
    );
    if (this.currentKeys.length == 0) return await this.maybe_close();
    this.stream.send({ channelKeys: this.currentKeys });
  }

  async read(): Promise<Sample[]> {
    if (!this.is_open) {
      throw new Error('Cannot read from a closed stream');
    }
    const [response, error] = await this.stream.receive();
    if (error) {
      throw error;
    }
    return response.samples;
  }
}
