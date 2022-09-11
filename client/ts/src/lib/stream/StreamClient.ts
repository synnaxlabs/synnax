import StreamReader from './StreamReader';
import Transport from '../transport';
import StreamWriter from './StreamWriter';

export type StreamClientProps = {
  transport: Transport;
};

export default class StreamClient {
  transport: Transport;
  reader: StreamReader;

  constructor({ transport }: StreamClientProps) {
    this.transport = transport;
    this.reader = new StreamReader({ transport: this.transport });
  }

  async newWriter(): Promise<StreamWriter> {
    const writer = new StreamWriter(this.transport.stream);
    await writer.open();
    return writer;
  }
}
