import { Endpoint } from '@synnaxlabs/freighter';
import Transport from './transport';

export type ClientProps = {
  host: string;
  port: number;
};

export default class Client {
  transport: Transport;

  constructor({ host, port }: ClientProps) {
    this.transport = new Transport(new Endpoint({ host, port }));
  }
}
