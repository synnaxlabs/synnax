// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import { useEffect } from "react";

import { HEARTBEAT } from "@/app/analytics/start";
import { Analytics } from "@/platform/analytics";

type Keys = Record<Analytics.Resource, Set<string>>;

/** The resources whose ontology entry is enough to count them. */
const PLAIN = [
  "device",
  "range",
  "schematic",
  "lineplot",
  "log",
  "table",
  "arc",
] as const satisfies Array<Analytics.Resource & ontology.ResourceType>;

type Plain = (typeof PLAIN)[number];

const isPlain = (type: ontology.ResourceType): type is Plain =>
  (PLAIN as readonly string[]).includes(type);

const toKeys = (keys: Array<string | number>): Set<string> => new Set(keys.map(String));

/**
 * Returns the keys of what the user has built. Internal channels and the Core's own
 * rack exist in every install, and a task counts once a running instance reports its
 * config, so a draft that was never deployed does not.
 */
const retrieve = async (client: Client): Promise<Keys> => {
  const [channels, racks, tasks, resources] = await Promise.all([
    client.channels.retrieve({ internal: false }),
    client.racks.retrieve({ embedded: false }),
    client.tasks.retrieve({ internal: false, snapshot: false, includeStatus: true }),
    client.ontology.retrieve({ types: [...PLAIN], excludeFieldData: true }),
  ]);
  const keys: Keys = {
    channel: toKeys(channels.map(({ key }) => key)),
    rack: toKeys(racks.map(({ key }) => key)),
    task: toKeys(
      tasks
        .filter(({ status }) => (status?.details.configHash ?? "") !== "")
        .map(({ key }) => key),
    ),
    device: new Set(),
    range: new Set(),
    schematic: new Set(),
    lineplot: new Set(),
    log: new Set(),
    table: new Set(),
    arc: new Set(),
  };
  resources.forEach(({ id }) => {
    if (!isPlain(id.type)) throw new Error(`retrieved a resource of type ${id.type}`);
    keys[id.type].add(id.key);
  });
  return keys;
};

const count = (keys: Keys): Analytics.Workspace => ({
  channel_count: keys.channel.size,
  device_count: keys.device.size,
  rack_count: keys.rack.size,
  task_count: keys.task.size,
  range_count: keys.range.size,
  schematic_count: keys.schematic.size,
  line_plot_count: keys.lineplot.size,
  log_count: keys.log.size,
  table_count: keys.table.size,
  arc_count: keys.arc.size,
});

interface WatchParams {
  client: Client;
  sink: Pick<Analytics.Sink, "capture" | "describe">;
}

/**
 * Counts what the user has built now and every {@link HEARTBEAT} until the returned
 * destructor is called. Each count reports one `resource_created` per kind of resource
 * that gained a key, whichever client created it, and describes the install whenever
 * a count changed.
 */
const watchWorkspace = ({ client, sink }: WatchParams): destructor.Destructor => {
  let known: Keys | null = null;
  let described: string | null = null;
  let stopped = false;
  const poll = async (): Promise<void> => {
    try {
      const next = await retrieve(client);
      if (stopped) return;
      const previous = known;
      if (previous != null)
        Analytics.resourceZ.options.forEach((resource) => {
          if ([...next[resource]].some((key) => !previous[resource].has(key)))
            sink.capture("resource_created", { resource });
        });
      known = next;
      const workspace = count(next);
      const serialized = JSON.stringify(workspace);
      if (serialized === described) return;
      described = serialized;
      sink.describe(workspace);
    } catch (err) {
      console.error("failed to count the workspace", err);
    }
  };
  void poll();
  const interval = setInterval(() => void poll(), HEARTBEAT.milliseconds);
  return () => {
    stopped = true;
    clearInterval(interval);
  };
};

/**
 * Watches what the user has built. It counts from its first successful poll, so it
 * mounts only where the Core already accepts requests.
 */
export const WatchWorkspace = (): null => {
  const sink = Analytics.use();
  const client = Synnax.use();
  useEffect(() => {
    if (client == null) return;
    return watchWorkspace({ client, sink });
  }, [client, sink]);
  return null;
};
