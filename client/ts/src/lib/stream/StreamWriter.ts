import { Sample } from './sample';
import { ClientStream, StreamClient } from '@arya-analytics/freighter';

type StreamWriterRequest = {
  samples: Sample[];
};

const endpoint = '/stream/write';

export default class StreamWriter {
  transport: StreamClient;
  stream: ClientStream<StreamWriterRequest, undefined>;

  constructor(transport: StreamClient) {
    this.transport = transport;
  }

  async open() {
    this.stream = await this.transport.stream<StreamWriterRequest, undefined>(
      endpoint
    );
  }

  async close() {
    this.stream.closeSend();
  }

  async write(samples: Sample[]) {
    this.stream.send({ samples });
  }
}
