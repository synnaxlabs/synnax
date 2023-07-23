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
import { ChannelClient, ChannelCreator } from "@/channel";
import { CacheChannelRetriever, ClusterChannelRetriever } from "@/channel/retriever";
import { Connectivity } from "@/connectivity";
import { FrameClient } from "@/framer";
import { OntologyClient } from "@/ontology";
import { RangeClient, RangeCreator, RangeRetriever } from "@/ranger";
import { Transport } from "@/transport";

export const synnaxPropsZ = z.object({
  host: z.string().min(1),
  port: z.number().or(z.string()),
  username: z.string().optional(),
  password: z.string().optional(),
  connectivityPollFrequency: TimeSpan.z.default(TimeSpan.seconds(30)),
  secure: z.boolean().optional().default(false),
});

export type SynnaxProps = z.input<typeof synnaxPropsZ>;
export type ParsedSynnaxProps = z.output<typeof synnaxPropsZ>;

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
  telem: FrameClient;
  ranges: RangeClient;
  channels: ChannelClient;
  auth: AuthenticationClient | undefined;
  connectivity: Connectivity;
  ontology: OntologyClient;
  props: ParsedSynnaxProps;
  static readonly connectivity = Connectivity;

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
  constructor(props: SynnaxProps) {
    this.props = synnaxPropsZ.parse(props);
    const { host, port, username, password, connectivityPollFrequency, secure } =
      this.props;
    this.transport = new Transport(new URL({ host, port: Number(port) }), secure);
    if (username != null && password != null) {
      this.auth = new AuthenticationClient(this.transport.unary, {
        username,
        password,
      });
      this.transport.use(this.auth.middleware());
    }
    const chRetriever = new CacheChannelRetriever(
      new ClusterChannelRetriever(this.transport.unary)
    );
    const chCreator = new ChannelCreator(this.transport.unary);
    this.telem = new FrameClient(this.transport.stream, chRetriever);
    this.channels = new ChannelClient(this.telem, chRetriever, chCreator);
    this.connectivity = new Connectivity(
      this.transport.unary,
      connectivityPollFrequency
    );
    this.ontology = new OntologyClient(this.transport.unary);
    const rangeRetriever = new RangeRetriever(this.transport.unary);
    const rangeCreator = new RangeCreator(this.transport.unary);
    this.ranges = new RangeClient(this.telem, rangeRetriever, rangeCreator);
  }

  close(): void {
    this.connectivity.stopChecking();
  }
}
