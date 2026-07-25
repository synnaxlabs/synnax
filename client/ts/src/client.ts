// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware } from "@synnaxlabs/freighter";
import {
  breaker,
  type CrudeTimeSpan,
  errors,
  TimeSpan,
  TimeStamp,
  url,
  uuid,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { access } from "@/access";
import { arc } from "@/arc";
import { auth } from "@/auth";
import { channel } from "@/channel";
import { connection } from "@/connection";
import { control } from "@/control";
import { device } from "@/device";
import { DisconnectedError, errorsMiddleware } from "@/errors";
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
  onInternalError: z
    .function({ input: z.tuple([z.instanceof(Error)]), output: z.void() })
    .optional(),
});

export interface SynnaxParams extends z.input<typeof synnaxParamsZ> {}
export interface ParsedSynnaxParams extends z.infer<typeof synnaxParamsZ> {}

/**
 * Client to perform operations against a Synnax cluster.
 *
 * @property channel - Channel client for creating and retrieving channels.
 * @property data - Data client for reading and writing telemetry.
 * @property connection - The connection state machine.
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
  /**
   * The connection state machine: one observable lifecycle over the heartbeat,
   * change-stream health, and auth outcomes. Read {@link connection.Machine.state}
   * or subscribe via {@link connection.Machine.onChange}.
   */
  readonly connection: connection.Machine;
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
   * A Synnax client must be closed when it is no longer needed. This stops the
   * connection machine and closes the change stream.
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
      retry: breaker,
    } = parsedParams;
    const transport = new Transport(
      new url.URL({ host, port: Number(port) }),
      breaker,
      secure,
    );
    transport.use(errorsMiddleware);
    const chRetriever = new channel.ClusterRetriever(transport.unary);
    super(transport.stream, transport.unary, chRetriever);
    const cache = new query.Cache({
      openStreamer: parsedParams.cache
        ? async (channels, { onOpen, onReopen }) => {
            const hardened = await framer.HardenedStreamer.open(
              (config) => this.openStreamer(config),
              channels,
              breaker,
              () => {
                onReopen?.();
                this.connection.notifyStreamLive();
              },
              (error) => this.connection.notifyStreamDrop(error),
            );
            // Reads start when the ObservableStreamer is constructed below,
            // so onOpen fires strictly before any frame or reconnect.
            onOpen?.();
            this.connection.notifyStreamLive();
            return new framer.ObservableStreamer(hardened);
          }
        : null,
      onError: parsedParams.onInternalError,
    });
    this.cache = cache;
    this.connection = new connection.Machine({
      unary: transport.unary,
      clientVersion: __VERSION__,
      name: parsedParams.name,
      heartbeatInterval: connectivityPollFrequency,
      clockSkewThreshold,
      retry: breaker,
      bringUpStream: parsedParams.cache
        ? async () => await cache.ensureStreaming()
        : undefined,
      onInternalError: parsedParams.onInternalError,
    });
    cache.onEpoch((epoch) => this.connection.ingestEpoch(epoch));
    transport.unary.use(this.shortCircuitMiddleware());
    this.auth = new auth.Client(
      transport.unary,
      { username, password },
      {
        onSuccess: () => this.connection.notifyAuthSuccess(),
        onFailure: (error) => this.connection.notifyAuthFailure(error),
      },
    );
    transport.use(this.auth.middleware());
    const chCreator = new channel.Writer(transport.unary);
    this.key = uuid.create();
    this.createdAt = TimeStamp.now();
    this.params = parsedParams;
    this.transport = transport;
    this.ontology = new ontology.Client(this.transport.unary, cache);
    const ontologyStores = this.ontology.stores;
    const rangeWriter = new ranger.Writer(this.transport.unary);
    this.labels = new label.Client(
      this.transport.unary,
      cache,
      ontologyStores.relationships,
    );
    this.statuses = new status.Client(
      this.transport.unary,
      cache,
      this.labels.store,
      ontologyStores,
      this.labels,
    );
    this.ranges = new ranger.Client(
      this,
      rangeWriter,
      this.transport.unary,
      chRetriever,
      this.labels,
      this.ontology,
      cache,
    );
    this.channels = new channel.Client(
      this,
      chRetriever,
      transport.unary,
      chCreator,
      this.statuses,
      this.ranges,
      cache,
      this.statuses.store,
      this.ranges.aliases,
      ontologyStores,
    );
    this.control = new control.Client(this);
    this.access = new access.Client(this.transport.unary, cache, ontologyStores);
    this.users = new user.Client(this.transport.unary, cache, ontologyStores);
    this.projects = new project.Client(this.transport.unary, cache, ontologyStores);
    this.tasks = new task.Client(
      this.transport.unary,
      this,
      this.ontology,
      this.ranges,
      cache,
      this.statuses.store,
    );
    this.racks = new rack.Client(
      this.transport.unary,
      this.tasks,
      cache,
      this.statuses.store,
      ontologyStores,
    );
    this.devices = new device.Client(
      this.transport.unary,
      cache,
      this.statuses.store,
      ontologyStores,
    );
    this.arcs = new arc.Client(
      this.transport.unary,
      this.transport.stream,
      this.ontology,
      this.tasks,
      cache,
      this.statuses.store,
    );
    this.views = new view.Client(this.transport.unary, cache, ontologyStores);
    this.schematics = new schematic.Client(
      this.transport.unary,
      this.ontology,
      cache,
      ontologyStores,
    );
    this.lineplots = new lineplot.Client(this.transport.unary, cache, ontologyStores);
    this.panels = new panel.Client(
      this.transport.unary,
      this.ontology,
      cache,
      ontologyStores,
    );
    this.logs = new log.Client(this.transport.unary, cache, ontologyStores);
    this.tables = new table.Client(this.transport.unary, cache, ontologyStores);
    this.groups = new group.Client(
      this.transport.unary,
      this.ontology,
      cache,
      ontologyStores,
    );
    this.imex = new imex.Client(this.transport.file);
    this.connection.start();
  }

  /**
   * Awaits the connection machine's first success. Idempotent: resolves
   * immediately when already connected.
   * @throws {AuthError} on a definitive credential rejection.
   * @throws {DisconnectedError | Unreachable} when the cluster cannot be
   * reached after the configured retry budget, or on timeout.
   */
  async connect(opts: ConnectOptions = {}): Promise<connection.State> {
    return await this.connection.awaitConnected(opts.timeout);
  }

  /** Supplies new credentials after an auth failure and resumes connecting. */
  reauthenticate(credentials: auth.Credentials): void {
    this.connection.reauthenticate(() => this.auth.setCredentials(credentials));
  }

  close(): void {
    this.connection.stop().catch(console.error);
    this.cache.close().catch(console.error);
  }

  /**
   * Rejects unary requests instantly while the machine knows the cluster is
   * unreachable, instead of burning the transport's retry budget per call.
   * The probe and login targets are exempt so the machine can heal.
   */
  private shortCircuitMiddleware(): Middleware {
    const EXEMPT = ["/connectivity/check", "/auth/login"];
    return async (ctx, next) => {
      const state = this.connection.state;
      if (
        state.variant === "error" &&
        state.reason === "unreachable" &&
        !EXEMPT.some((target) => ctx.target.endsWith(target))
      )
        throw new DisconnectedError(
          `Cannot reach cluster at ${this.params.host}:${this.params.port}`,
        );
      return await next(ctx);
    };
  }
}

export interface ConnectOptions {
  /**
   * Maximum time to wait before rejecting. Without it, connect rejects when
   * the machine escalates to error(unreachable) after its retry budget.
   */
  timeout?: CrudeTimeSpan;
}

/**
 * Constructs a client and awaits its first successful connection.
 * On rejection the constructed client is closed before the error propagates.
 * @throws see {@link Synnax.connect}.
 */
export const connect = async (
  params: SynnaxParams,
  opts?: ConnectOptions,
): Promise<Synnax> => {
  const client = new Synnax(params);
  try {
    await client.connect(opts);
  } catch (err) {
    client.close();
    throw errors.fromUnknown(err);
  }
  return client;
};

export interface CheckConnectionParams extends Pick<
  SynnaxParams,
  "host" | "port" | "secure" | "retry" | "name"
> {}

/** Runs a one-shot connectivity probe against the given cluster address. */
export const checkConnection = async (
  params: CheckConnectionParams,
): Promise<connection.State> => {
  const { host, port, secure, name, retry } = params;
  const retryConfig = zod.parse(breaker.breakerConfigZ.optional(), retry);
  const endpoint = new url.URL({ host, port: Number(port) });
  const transport = new Transport(endpoint, retryConfig, secure);
  transport.use(errorsMiddleware);
  const machine = new connection.Machine({
    unary: transport.unary,
    clientVersion: __VERSION__,
    name,
    escalateAfter: 1,
  });
  return await machine.check();
};
