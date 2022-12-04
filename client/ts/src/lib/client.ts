import { URL } from '@synnaxlabs/freighter';
import { z } from 'zod';

import AuthenticationClient from './auth';
import ChannelCreator from './channel/creator';
import Registry from './channel/registry';
import ChannelRetriever from './channel/retriever';
import ConnectivityClient from './connectivity';
import FrameClient from './framer/client';
import { TimeSpan } from './telem';
import Transport from './transport';
import OntologyClient from './ontology/client';
import { ChannelClient } from './channel';

export const synnaxPropsSchema = z.object({
  host: z.string().min(1),
  port: z.number().or(z.string()),
  username: z.string().optional(),
  password: z.string().optional(),
  connectivityPollFrequency: z.instanceof(TimeSpan).optional(),
});

export type SynnaxProps = z.infer<typeof synnaxPropsSchema>;

/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 */
export default class Synnax {
  private transport: Transport;
  data: FrameClient;
  channel: ChannelClient;
  auth: AuthenticationClient | undefined;
  connectivity: ConnectivityClient;
  ontology: OntologyClient;

  /**
   * @param props.host - Hostname of a node in the cluster.
   * @param props.port - Port of the node in the cluster.
   * @param props.username - Username for authentication. Not required if the
   *   cluster is insecure.
   * @param props.password - Password for authentication. Not required if the
   *   cluster is insecure.
   */
  constructor({
    host,
    port,
    username,
    password,
    connectivityPollFrequency,
  }: SynnaxProps) {
    this.transport = new Transport(new URL({ host, port: Number(port) }));
    if (username && password) {
      this.auth = new AuthenticationClient(this.transport.httpFactory, {
        username,
        password,
      });
      this.transport.use(this.auth.middleware());
    }
    const chRetriever = new ChannelRetriever(this.transport);
    const chCreator = new ChannelCreator(this.transport);
    const chRegistry = new Registry(chRetriever);
    this.data = new FrameClient(this.transport, chRegistry);
    this.channel = new ChannelClient(this.data, chRetriever, chCreator);
    this.connectivity = new ConnectivityClient(
      this.transport.getClient(),
      connectivityPollFrequency
    );
    this.ontology = new OntologyClient(this.transport);
  }

  close() {
    this.connectivity.stopChecking();
  }
}
