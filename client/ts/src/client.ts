// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { breaker, TimeSpan, TimeStamp, url, uuid, zod } from "@synnaxlabs/x";
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
import { group } from "@/group";
import { imex } from "@/imex";
import { label } from "@/label";
import { lineplot } from "@/lineplot";
import { log } from "@/log";
import { ontology } from "@/ontology";
import { panel } from "@/panel";
import { project } from "@/project";
import { query } from "@/query";
import { rack } from "@/rack";
import { ranger } from "@/ranger";
import { schematic } from "@/schematic";
import { status } from "@/status";
import { table } from "@/table";
import { task } from "@/task";
import { Transport } from "@/transport";
import { user } from "@/user";
import { view } from "@/view";

export const synnaxParamsZ = z.object({
  host: z.string({ error: "Host is required" }).min(1, "Host is required"),
  port: z
    .number({ error: "Port is required" })
    .or(z.string({ error: "Port is required" })),
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
  connectivityPollFrequency: TimeSpan.z.default(TimeSpan.seconds(30)),
  clockSkewThreshold: TimeSpan.z.default(TimeSpan.seconds(1)),
  secure: z.boolean().default(false),
  name: z.string().optional(),
  retry: breaker.breakerConfigZ.optional(),
  cache: z.boolean().default(true),
  /**
   * Receives cache errors that have no caller to throw to (listener fan-out,
   * streamer frame handling, background reconciliation). Defaults to console
   * logging.
   */
  // output is unknown, not void: strict void validation would make a
  // caller's `(e) => list.push(e)` throw at error-report time
  onInternalError: z
    .function({ input: z.tuple([z.instanceof(Error)]), output: z.unknown() })
    .optional(),
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
  readonly key: string;
  readonly createdAt: TimeStamp;
  readonly params: ParsedSynnaxParams;
  readonly ranges: ranger.Client;
  readonly channels: channel.Client;
  readonly auth: auth.Client;
  readonly users: user.Client;
  readonly access: access.Client;
  readonly connectivity: connection.Checker;
  readonly ontology: ontology.Client;
  readonly projects: project.Client;
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
  readonly panels: panel.Client;
  readonly logs: log.Client;
  readonly tables: table.Client;
  readonly groups: group.Client;
  readonly imex: imex.Client;
  /**
   * The client's local mirror of cluster state. Live and streamed when
   * `cache: true` (the default); detached (local-only, no streaming) when
   * `cache: false`. Not a data access path: per-domain stores on the domain
   * clients remain the only way to read cached records.
   */
  readonly cache: query.Cache;
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
   * @param props.connectivityPollFrequency - Heartbeat cadence while the
   * connection is healthy. Defaults to 30 seconds.
   * @param props.secure - Whether to connect to the cluster using TLS. The cluster
   * must be configured to support TLS. Defaults to false.
   *
   * A Synnax client must be closed when it is no longer needed. This will stop
   * the client from polling the cluster for connectivity information.
   */
  constructor(params: SynnaxParams) {
    const parsedParams = zod.parse(synnaxParamsZ, params);
    const {
      host,
      port,
      username,
      password,
      connectivityPollFrequency,
      clockSkewThreshold,
      secure,
      retry,
    } = parsedParams;
    const transport = new Transport(
      new url.URL({ host, port: Number(port) }),
      retry,
      secure,
    );
    transport.use(errorsMiddleware);
    const chRetriever = new channel.ClusterRetriever(transport.unary);
    super({ stream: transport.stream, unary: transport.unary, retriever: chRetriever });
    const cache = new query.Cache({
      openStreamer: parsedParams.cache
        ? async (channels, { onOpen, onReopen }) => {
            const hardened = await framer.HardenedStreamer.open(
              (config) => this.openStreamer(config),
              channels,
              retry,
              () => onReopen?.(),
            );
            // Reads start when the ObservableStreamer is constructed below,
            // so onOpen fires strictly before any frame or reconnect.
            onOpen?.();
            return new framer.ObservableStreamer(hardened);
          }
        : null,
      onError: parsedParams.onInternalError,
    });
    this.cache = cache;
    this.connectivity = new connection.Checker(
      transport.unary,
      connectivityPollFrequency,
      this.clientVersion,
      parsedParams.name,
      clockSkewThreshold,
    );
    this.auth = new auth.Client(transport.unary, { username, password });
    transport.use(this.auth.middleware());
    const chCreator = new channel.Writer(transport.unary);
    this.key = uuid.create();
    this.createdAt = TimeStamp.now();
    this.params = parsedParams;
    this.transport = transport;
    const unary = this.transport.unary;
    this.ontology = new ontology.Client({ unary, cache });
    const rangeWriter = new ranger.Writer(unary);
    this.labels = new label.Client({ unary, cache, ontology: this.ontology });
    this.statuses = new status.Client({
      unary,
      cache,
      ontology: this.ontology,
      labels: this.labels,
    });
    this.ranges = new ranger.Client({
      framer: this,
      writer: rangeWriter,
      unary,
      channels: chRetriever,
      labels: this.labels,
      ontology: this.ontology,
      cache,
    });
    this.channels = new channel.Client({
      framer: this,
      retriever: chRetriever,
      unary,
      writer: chCreator,
      statuses: this.statuses,
      ranges: this.ranges,
      cache,
      ontology: this.ontology,
    });
    this.control = new control.Client({ framer: this });
    this.access = new access.Client({ unary, cache, ontology: this.ontology });
    this.users = new user.Client({ unary, cache, ontology: this.ontology });
    this.projects = new project.Client({ unary, cache, ontology: this.ontology });
    this.tasks = new task.Client({
      unary,
      framer: this,
      ontology: this.ontology,
      ranges: this.ranges,
      cache,
      statusStore: this.statuses.store,
    });
    this.racks = new rack.Client({
      unary,
      tasks: this.tasks,
      cache,
      statusStore: this.statuses.store,
      ontology: this.ontology,
    });
    this.devices = new device.Client({
      unary,
      cache,
      statusStore: this.statuses.store,
      ontology: this.ontology,
    });
    this.arcs = new arc.Client({
      unary,
      stream: this.transport.stream,
      ontology: this.ontology,
      tasks: this.tasks,
      cache,
      statusStore: this.statuses.store,
    });
    this.views = new view.Client({ unary, cache, ontology: this.ontology });
    this.schematics = new schematic.Client({
      unary,
      ontology: this.ontology,
      cache,
    });
    this.lineplots = new lineplot.Client({ unary, cache, ontology: this.ontology });
    this.panels = new panel.Client({
      unary,
      ontology: this.ontology,
      cache,
    });
    this.logs = new log.Client({ unary, cache, ontology: this.ontology });
    this.tables = new table.Client({ unary, cache, ontology: this.ontology });
    this.groups = new group.Client({
      unary,
      ontology: this.ontology,
      cache,
    });
    this.imex = new imex.Client({ file: this.transport.file });
  }

  close(): void {
    this.connectivity.stop();
    this.cache.close().catch(console.error);
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
  const retryConfig = zod.parse(breaker.breakerConfigZ.optional(), retry);
  const endpoint = new url.URL({ host, port: Number(port) });
  const transport = new Transport(endpoint, retryConfig, secure);
  return new connection.Checker(transport.unary, undefined, __VERSION__, name);
};
