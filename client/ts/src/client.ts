// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { breaker, TimeSpan, TimeStamp, URL } from "@synnaxlabs/x";
import { z } from "zod";

import { access } from "@/access";
import { arc } from "@/arc";
import { auth } from "@/auth";
import { channel } from "@/channel";
import { connection } from "@/connection";
import { control } from "@/control";
import { device } from "@/device";
import { errorsMiddleware } from "@/errors";
import { framer } from "@/framer";
import { label } from "@/label";
import { lineplot } from "@/lineplot";
import { log } from "@/log";
import { ontology } from "@/ontology";
import { rack } from "@/rack";
import { ranger } from "@/range";
import { schematic } from "@/schematic";
import { status } from "@/status";
import { table } from "@/table";
import { task } from "@/task";
import { Transport } from "@/transport";
import { user } from "@/user";
import { view } from "@/view";
import { workspace } from "@/workspace";

export const synnaxParamsZ = z.object({
  host: z.string({ error: "Host is required" }).min(1, "Host is required"),
  port: z
    .number({ error: "Port is required" })
    .or(z.string({ error: "Port is required" })),
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
  connectivityPollFrequency: TimeSpan.z.default(TimeSpan.seconds(30)),
  secure: z.boolean().default(false),
  name: z.string().optional(),
  retry: breaker.breakerConfigZ.optional(),
});

export interface SynnaxParams extends z.input<typeof synnaxParamsZ> {}
export interface ParsedSynnaxParams extends z.infer<typeof synnaxParamsZ> {}

/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 * @property connectivity - Client for retrieving connectivity information.
 * @property ontology - Client for querying the cluster's ontology.
 */
export default class Synnax extends framer.Client {
  readonly createdAt: TimeStamp;
  readonly params: ParsedSynnaxParams;
  readonly ranges: ranger.Client;
  readonly channels: channel.Client;
  readonly auth: auth.Client;
  readonly users: user.Client;
  readonly access: access.Client;
  readonly connectivity: connection.Checker;
  readonly ontology: ontology.Client;
  readonly workspaces: workspace.Client;
  readonly labels: label.Client;
  readonly statuses: status.Client;
  readonly tasks: task.Client;
  readonly racks: rack.Client;
  readonly devices: device.Client;
  readonly control: control.Client;
  readonly arcs: arc.Client;
  readonly views: view.Client;
  readonly schematics: schematic.Client;
  readonly lineplots: lineplot.Client;
  readonly logs: log.Client;
  readonly tables: table.Client;
  static readonly connectivity = connection.Checker;
  private readonly transport: Transport;

  /**
   * The version of the client.
   */
  readonly clientVersion: string = __VERSION__;

  /**
   * @param props.host - Hostname of a node in the cluster.
   * @param props.port - Port of the node in the cluster.
   * @param props.username - Username for authentication. Not required if the
   * cluster is insecure.
   * @param props.password - Password for authentication. Not required if the
   * cluster is insecure.
   * @param props.connectivityPollFrequency - Frequency at which to poll the
   * cluster for connectivity information. Defaults to 30 seconds.
   * @param props.secure - Whether to connect to the cluster using TLS. The cluster
   * must be configured to support TLS. Defaults to false.
   *
   * A Synnax client must be closed when it is no longer needed. This will stop
   * the client from polling the cluster for connectivity information.
   */
  constructor(params: SynnaxParams) {
    const parsedParams = synnaxParamsZ.parse(params);
    const {
      host,
      port,
      username,
      password,
      connectivityPollFrequency,
      secure,
      retry: breaker,
    } = parsedParams;
    const transport = new Transport(
      new URL({ host, port: Number(port) }),
      breaker,
      secure,
    );
    transport.use(errorsMiddleware);
    const chRetriever = new channel.CacheRetriever(
      new channel.ClusterRetriever(transport.unary),
    );
    super(transport.stream, transport.unary, chRetriever);
    this.auth = new auth.Client(transport.unary, { username, password });
    transport.use(this.auth.middleware());
    const chCreator = new channel.Writer(transport.unary, chRetriever);
    this.createdAt = TimeStamp.now();
    this.params = parsedParams;
    this.transport = transport;
    this.channels = new channel.Client(this, chRetriever, transport.unary, chCreator);
    this.connectivity = new connection.Checker(
      transport.unary,
      connectivityPollFrequency,
      this.clientVersion,
      parsedParams.name,
    );
    this.control = new control.Client(this);
    this.ontology = new ontology.Client(transport.unary, this);
    const rangeWriter = new ranger.Writer(this.transport.unary);
    this.labels = new label.Client(this.transport.unary);
    this.statuses = new status.Client(this.transport.unary);
    this.ranges = new ranger.Client(
      this,
      rangeWriter,
      this.transport.unary,
      chRetriever,
      this.labels,
      this.ontology,
    );
    this.access = new access.Client(this.transport.unary);
    this.users = new user.Client(this.transport.unary);
    this.workspaces = new workspace.Client(this.transport.unary);
    this.tasks = new task.Client(
      this.transport.unary,
      this,
      this.ontology,
      this.ranges,
    );
    this.racks = new rack.Client(this.transport.unary, this.tasks);
    this.devices = new device.Client(this.transport.unary);
    this.arcs = new arc.Client(this.transport.unary, this.transport.stream);
    this.views = new view.Client(this.transport.unary);
    this.schematics = new schematic.Client(this.transport.unary);
    this.lineplots = new lineplot.Client(this.transport.unary);
    this.logs = new log.Client(this.transport.unary);
    this.tables = new table.Client(this.transport.unary);
  }

  get key(): string {
    return this.createdAt.valueOf().toString();
  }

  close(): void {
    this.connectivity.stop();
  }
}

export interface CheckConnectionParams extends Pick<
  SynnaxParams,
  "host" | "port" | "secure" | "retry" | "name"
> {}

export const checkConnection = async (params: CheckConnectionParams) =>
  await newConnectionChecker(params).check();

export const newConnectionChecker = (params: CheckConnectionParams) => {
  const { host, port, secure, name, retry } = params;
  const retryConfig = breaker.breakerConfigZ.optional().parse(retry);
  const url = new URL({ host, port: Number(port) });
  const transport = new Transport(url, retryConfig, secure);
  return new connection.Checker(transport.unary, undefined, __VERSION__, name);
};
