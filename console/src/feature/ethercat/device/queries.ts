// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type rack, status, type Synnax, type task } from "@synnaxlabs/client";
import { Device, Flux, Task as PTask } from "@synnaxlabs/pluto";
import { array, primitive, TimeSpan, unique, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { SLAVE_SCHEMAS, type SlaveDevice } from "@/feature/ethercat/device/types";
import {
  type Channel,
  READ_TYPE,
  SCAN_TYPE,
  WRITE_TYPE,
} from "@/feature/ethercat/task/types";
import { Task } from "@/platform/task";

export const { useResult: useResultSlave } = Device.createRetrieve(SLAVE_SCHEMAS);

const { use: useSlaves, useResult: useResultSlaves } =
  Device.createRetrieveMultiple(SLAVE_SCHEMAS);

/** The slaves the keys name, or undefined until they resolve or on failure. */
export const useSlavesByKeys = (keys: device.Key[]): SlaveDevice[] | undefined =>
  useResultSlaves(keys.length === 0 ? null : { where: { key: { in: keys } } }).data;

export interface EnabledState {
  allEnabled: boolean;
  allDisabled: boolean;
}

export interface EnabledStateParams {
  keys: device.Key[];
}

export const useEnabledState = ({ keys }: EnabledStateParams): EnabledState => {
  const slaves = useSlaves({ where: { key: { in: keys } } });
  const disabledCount = slaves.filter((d) => !d.properties?.enabled).length;
  return {
    allDisabled: disabledCount === slaves.length,
    allEnabled: disabledCount === 0,
  };
};

export const useCommonNetwork = (channels: Channel[]) => {
  const keys = channels.map((ch) => ch.device).filter((c) => c != null);
  const firstDeviceKey = keys.length > 0 ? keys[0] : "";
  const { data: slave } = useResultSlave(
    primitive.isZero(firstDeviceKey) ? null : { key: firstDeviceKey },
  );
  return slave?.properties?.network ?? "";
};

export interface ToggleEnabledParams {
  keys: device.Key | device.Key[];
  enabled?: boolean;
}

export const { useUpdate: useToggleEnabled } = Flux.createUpdate<ToggleEnabledParams>({
  name: "toggle enabled",
  verbs: verbs.UPDATE,
  update: async ({ data, client }) => {
    const keys = array.toArray(data.keys);
    let { enabled } = data;
    if (enabled == null) {
      const first = await client.devices.retrieve({
        key: keys[0],
        schemas: SLAVE_SCHEMAS,
      });
      enabled = !first.properties.enabled;
    }
    const set = device.set({ path: ["properties", "enabled"], value: enabled });
    await Promise.all(keys.map(async (key) => await client.devices.dispatch(key, set)));
    return data;
  },
});

/** The scan task command that switches a slave to Synnax only. */
const SET_STANDALONE_COMMAND = "set_standalone";

const SWITCH: verbs.Verbs = {
  present: "switch",
  past: "switched",
  participle: "switching",
};

const channelDevicesZ = z.object({
  channels: z.array(z.object({ device: z.string() })).default(() => []),
});

const devicesOf = (t: task.Task): device.Key[] =>
  unique.unique(channelDevicesZ.parse(t.config).channels.map((ch) => ch.device));

const isRunning = (t: task.Task): boolean =>
  (t.type === READ_TYPE || t.type === WRITE_TYPE) && t.status?.details.running === true;

const onNetwork = (
  tasks: task.Task[],
  slaves: SlaveDevice[],
  network: string,
): task.Task[] => {
  const keys = new Set(
    slaves.filter((s) => s.properties.network === network).map((s) => s.key),
  );
  return tasks.filter((t) => devicesOf(t).some((k) => keys.has(k)));
};

/** @returns the EtherCAT read and write tasks that run on the network of the rack. */
const retrieveRunningOnNetwork = async (
  client: Synnax,
  rackKey: rack.Key,
  network: string,
): Promise<task.Task[]> => {
  const running = (
    await client.tasks.retrieve({ rack: rackKey, includeStatus: true })
  ).filter(isRunning);
  const keys = unique.unique(running.flatMap(devicesOf));
  const slaves =
    keys.length === 0
      ? []
      : await client.devices.retrieve({ key: keys, schemas: SLAVE_SCHEMAS });
  return onNetwork(running, slaves, network);
};

/** The EtherCAT read and write tasks that run on the network of the rack, kept live. */
export const useRunningOnNetwork = (
  rackKey: rack.Key,
  network: string,
): task.Task[] => {
  const tasks =
    PTask.useResultMultiple({ where: { rack: { eq: rackKey } } }).data ?? [];
  const running = tasks.filter(isRunning);
  const slaves = useSlavesByKeys(unique.unique(running.flatMap(devicesOf))) ?? [];
  return onNetwork(running, slaves, network);
};

export interface SetStandaloneParams {
  slave: SlaveDevice;
}

/**
 * Stops each EtherCAT task that runs on the network of the slave, then switches the
 * slave to Synnax only. The slave changes after its power restarts. When a stop or the
 * switch fails, starts the tasks again.
 */
export const { useUpdate: useSetStandalone } = Flux.createUpdate<SetStandaloneParams>({
  name: "EtherCAT module",
  verbs: SWITCH,
  update: async ({ client, data }) => {
    const { rack, key, properties } = data.slave;
    const tasks = await retrieveRunningOnNetwork(client, rack, properties.network);
    await Task.whileStopped({ tasks, keepStopped: true }, async () => {
      const scanTask = await client.tasks.retrieve({
        type: SCAN_TYPE,
        rack,
        one: true,
      });
      const res = await scanTask.executeCommandSync({
        type: SET_STANDALONE_COMMAND,
        args: { device: key },
        timeout: TimeSpan.seconds(30),
      });
      if (res.variant === "error") throw status.toError(res);
    });
    return data;
  },
});
