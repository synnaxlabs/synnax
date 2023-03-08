// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, URL } from "@synnaxlabs/x";
import { z } from "zod";

import { AuthenticationClient } from "@/auth";
import { ChannelClient } from "@/channel";
import { ConnectivityClient } from "@/connectivity";
import { FrameClient } from "@/framer";
import { OntologyClient } from "@/ontology";
import { Transport } from "@/transport";

export const synnaxPropsSchema = z.object({
  host: z.string().min(1),
  port: z.number().or(z.string()),
  username: z.string().optional(),
  password: z.string().optional(),
  connectivityPollFrequency: z.instanceof(TimeSpan).optional(),
  secure: z.boolean().default(false).optional(),
});

export type SynnaxProps = z.infer<typeof synnaxPropsSchema>;

/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 * @property connectivity - Client for retrieving connectivity information.
 * @property ontology - Client for querying the cluster's ontology.
 */
// eslint-disable-next-line import/no-default-export
export default class Synnax {
  private readonly transport: Transport;
  data: FrameClient;
  channels: ChannelClient;
  auth: AuthenticationClient | undefined;
  connectivity: ConnectivityClient;
  ontology: OntologyClient;

  /**
   * @param props.host - Hostname of a node in the cluster.
   * @param props.port - Port of the node in the cluster.
   * @param props.username - Username for authentication. Not required if the
   * cluster is insecure.
   * @param props.password - Password for authentication. Not required if the
   * cluster is insecure.
   * @param props.connectivityPollFrequency - Frequency at which to poll the
   * cluster for connectivity information. Defaults to 5 seconds.
   * @param props.secure - Whether to connect to the cluster using TLS. The cluster
   * must be configured to support TLS. Defaults to false.
   *
   * A Synnax client must be closed when it is no longer needed. This will stop
   * the client from polling the cluster for connectivity information.
   */
  constructor({
    host,
    port,
    username,
    password,
    connectivityPollFrequency,
    secure,
  }: SynnaxProps) {
    this.transport = new Transport(new URL({ host, port: Number(port) }), secure);
    if (username != null && password != null) {
      this.auth = new AuthenticationClient(this.transport.httpFactory, {
        username,
        password,
      });
      this.transport.use(this.auth.middleware());
    }
    this.data = new FrameClient(this.transport);
    this.channels = new ChannelClient(this.data, this.transport);
    this.connectivity = new ConnectivityClient(
      this.transport.getClient(),
      connectivityPollFrequency
    );
    this.ontology = new OntologyClient(this.transport);
  }

  close(): void {
    this.connectivity.stopChecking();
  }
}
