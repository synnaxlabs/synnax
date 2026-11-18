// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type rack, status, type Synnax, type task } from "@synnaxlabs/client";
import { Device, Flux, Task } from "@synnaxlabs/pluto";
import { array, deep, primitive, TimeSpan, unique, verbs } from "@synnaxlabs/x";

import { canReplace } from "@/feature/ethercat/device/replace";
import {
  MAKE,
  SLAVE_MODEL,
  SLAVE_SCHEMAS,
  type SlaveDevice,
  type SlaveOnRack,
  toSlaveOnRack,
} from "@/feature/ethercat/device/types";
import {
  type Channel,
  devicesOf,
  READ_TYPE,
  SCAN_TYPE,
  WRITE_TYPE,
} from "@/feature/ethercat/task/types";
import { Device as PlatformDevice } from "@/platform/device";

export const { useResult: useResultSlave } = Device.createRetrieve(SLAVE_SCHEMAS);

const {
  use: useSlaves,
  useResult: useResultSlaves,
  createResultSelector,
} = Device.createRetrieveMultiple(SLAVE_SCHEMAS);

/** The slaves the keys name, or undefined until they resolve or on failure. */
export const useSlavesByKeys = (keys: device.Key[]): SlaveDevice[] | undefined =>
  useResultSlaves(keys.length === 0 ? null : { where: { key: { in: keys } } }).data;

// A status heartbeat changes no slave unless it connects or disconnects.
const useResultSlavesOnRack = createResultSelector(
  (slaves) => slaves.map(toSlaveOnRack),
  deep.equal,
);

/**
 * The slaves on the rack, or undefined until they resolve, on failure, or without a
 * rack.
 */
export const useSlavesOnRack = (rack: number | undefined): SlaveOnRack[] | undefined =>
  useResultSlavesOnRack(
    rack == null
      ? null
      : {
          where: { make: { eq: MAKE }, model: { eq: SLAVE_MODEL }, rack: { eq: rack } },
          includeStatus: true,
        },
  ).data;

/** Every slave, or undefined until they resolve or on failure. */
export const useEverySlave = (): SlaveOnRack[] | undefined =>
  useResultSlavesOnRack({
    where: { make: { eq: MAKE }, model: { eq: SLAVE_MODEL } },
    includeStatus: true,
  }).data;

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

export interface SetStandaloneParams {
  slave: SlaveDevice;
}

export interface SetStandaloneResult extends SetStandaloneParams {
  /** The answer of the Driver to the switch, which says what the user does next. */
  answer: task.Status;
}

/**
 * How long a switch waits for the Driver. Writes to several slaves can be slow. It ends
 * before the default 60 s operation timeout of the Driver, whose error reply would read
 * as a refusal while the switch still runs.
 */
const SWITCH_TIMEOUT = TimeSpan.seconds(50);

const startTasks = async (tasks: task.Task[]): Promise<void> => {
  // A task that fails to start shows its own error.
  await Promise.allSettled(
    tasks.map(async (t) => await t.executeCommandSync({ type: "start" })),
  );
};

const isRunning = (t: task.Task): boolean =>
  (t.type === READ_TYPE || t.type === WRITE_TYPE) && t.status?.details.running === true;

/** @returns the tasks with a channel on a slave of the network. */
const onNetwork = (
  tasks: task.Task[],
  slaves: SlaveDevice[],
  network: string,
): task.Task[] => {
  const keys = new Set(
    slaves.filter((s) => s.properties.network === network).map((s) => s.key),
  );
  return tasks.filter((t) => devicesOf(t.config).some((k) => keys.has(k)));
};

/**
 * Stops each EtherCAT read and write task that runs on the network of the rack. When a
 * stop fails, starts the tasks again.
 * @returns the stopped tasks.
 * @throws {Error} if a task fails to stop.
 */
const stopTasksOnNetwork = async (
  client: Synnax,
  rack: rack.Key,
  network: string,
): Promise<task.Task[]> => {
  const running = (await client.tasks.retrieve({ rack, includeStatus: true })).filter(
    isRunning,
  );
  const keys = unique.unique(running.flatMap((t) => devicesOf(t.config)));
  const slaves =
    keys.length === 0
      ? []
      : await client.devices.retrieve({ key: keys, schemas: SLAVE_SCHEMAS });
  const tasks = onNetwork(running, slaves, network);
  let stopped = false;
  try {
    await Promise.all(
      tasks.map(async (t) => {
        const res = await t.executeCommandSync({ type: "stop" });
        if (res.variant === "error") throw status.toError(res);
      }),
    );
    stopped = true;
  } finally {
    if (!stopped) await startTasks(tasks);
  }
  return tasks;
};

/** The EtherCAT read and write tasks that run on the network of the rack, kept live. */
export const useRunningOnNetwork = (
  rackKey: rack.Key,
  network: string,
): task.Task[] => {
  const tasks =
    Task.useResultMultiple({
      where: { rack: { eq: rackKey }, type: { in: [READ_TYPE, WRITE_TYPE] } },
    }).data ?? [];
  const running = tasks.filter(isRunning);
  const slaves =
    useSlavesByKeys(unique.unique(running.flatMap((t) => devicesOf(t.config)))) ?? [];
  return onNetwork(running, slaves, network);
};

const RESTART: verbs.Verbs = {
  present: "restart",
  past: "restarted",
  participle: "restarting",
};

export interface RestartTasksParams {
  rack: rack.Key;
  network: string;
}

/**
 * Stops each EtherCAT read and write task that runs on the network of the rack, then
 * starts the tasks again, so that the network starts with the saved settings of its
 * slaves. A task that fails to start shows its own error.
 * @throws {Error} if a task fails to stop, after the tasks start again.
 */
export const { useUpdate: useRestartTasks } = Flux.createUpdate<RestartTasksParams>({
  name: "tasks",
  verbs: RESTART,
  update: async ({ client, data }) => {
    await startTasks(await stopTasksOnNetwork(client, data.rack, data.network));
    return data;
  },
});

/**
 * Stops each EtherCAT task that runs on the network of the slave, then switches the
 * slave to Synnax only. When a stop fails or the Driver refuses the switch, starts the
 * tasks again. Else the tasks stay stopped, and the answer tells the user to start
 * them.
 */
export const { useUpdate: useSetStandalone } = Flux.createUpdate<
  SetStandaloneParams,
  SetStandaloneResult
>({
  name: "EtherCAT module",
  verbs: SWITCH,
  update: async ({ client, data }) => {
    const { rack, key } = data.slave;
    const scanTask = await client.tasks.retrieve({ type: SCAN_TYPE, rack, one: true });
    const tasks = await stopTasksOnNetwork(client, rack, data.slave.properties.network);
    const res = await scanTask
      .executeCommandSync({
        type: SET_STANDALONE_COMMAND,
        args: { device: key },
        timeout: SWITCH_TIMEOUT,
      })
      .catch((e: unknown) => {
        const then =
          tasks.length === 0 ? "" : " Start your tasks again after you check it.";
        throw new Error(`The switch may still be running. Check the device.${then}`, {
          cause: e,
        });
      });
    if (res.variant === "error") {
      await startTasks(tasks);
      throw status.toError(res);
    }
    // The scan task writes the properties and status of each device at a scan. It also
    // scans every 5 s, so a failed scan command only delays them.
    await scanTask
      .executeCommand({ type: PlatformDevice.SCAN_COMMAND })
      .catch(() => undefined);
    if (tasks.length === 0) return { ...data, answer: res };
    const description = `${res.description} Start your tasks again.`.trimStart();
    return { ...data, answer: { ...res, description } };
  },
});

/** The scan task command that turns off the DewesoftX filter. */
const DISABLE_DEWESOFT_FILTER_COMMAND = "disable_dewesoft_filter";

const TURN_OFF: verbs.Verbs = {
  present: "turn off",
  past: "turned off",
  participle: "turning off",
};

export interface DisableFilterParams {
  /** The scan task of the rack whose Driver turns off the filter. */
  scanner: task.Task;
}

/**
 * Starts to turn off the DewesoftX filter on the adapters that the last scan found it
 * blocks, and returns the answer of the Driver. Windows can ask for administrator
 * access on the computer that runs the Driver, and the next scans show whether the
 * filter is off.
 */
export const { useUpdate: useDisableFilter } = Flux.createUpdate<
  DisableFilterParams,
  task.Status
>({
  name: "the DewesoftX filter",
  verbs: TURN_OFF,
  update: async ({ data }) => {
    const res = await data.scanner.executeCommandSync({
      type: DISABLE_DEWESOFT_FILTER_COMMAND,
    });
    if (res.variant === "error") throw status.toError(res);
    return res;
  },
});

export interface ReplaceParams {
  /** The disconnected slave that keeps its key, name, settings, and tasks. */
  old: device.Key;
  /** The new slave whose module the old slave takes. The replace deletes it. */
  slave: device.Key;
}

export interface ReplaceResult {
  /** The old slave as the replace found it. */
  old: SlaveDevice;
  /** The EtherCAT tasks that the replace stopped. They stay stopped. */
  stopped: task.Task[];
}

const REPLACE: verbs.Verbs = {
  present: "replace",
  past: "replaced",
  participle: "replacing",
};

/**
 * Gives the old slave the serial number, network, and position of the new slave, so
 * the next scan finds the module as the old slave. Then deletes the new slave. Stops
 * each EtherCAT task that runs on the network first, and leaves the tasks stopped.
 * @throws {Error} if the new slave can no longer replace the old slave, a task fails to
 * stop, or the write to the old slave fails. The slaves then stay as they are, and the
 * stopped tasks start again.
 * @throws {Error} if the delete of the new slave fails. The error tells the user to
 * delete the new slave and start the stopped tasks.
 */
export const { useUpdate: useReplace } = Flux.createUpdate<
  ReplaceParams,
  ReplaceResult
>({
  name: "EtherCAT module",
  verbs: REPLACE,
  update: async ({ client, data }) => {
    const [old, slave] = await Promise.all(
      [data.old, data.slave].map(
        async (key) =>
          await client.devices.retrieve({
            key,
            schemas: SLAVE_SCHEMAS,
            includeStatus: true,
          }),
      ),
    );
    if (!canReplace(toSlaveOnRack(old), toSlaveOnRack(slave)))
      throw new Error(`${slave.name} can no longer take the place of ${old.name}.`);
    const stopped = await stopTasksOnNetwork(client, old.rack, old.properties.network);
    // A scan between the two writes would make the new slave again if the delete went
    // first.
    let moved = false;
    try {
      await client.devices.dispatch(old.key, [
        device.set({ path: ["properties", "serial"], value: slave.properties.serial }),
        device.set({
          path: ["properties", "network"],
          value: slave.properties.network,
        }),
        device.set({
          path: ["properties", "position"],
          value: slave.properties.position,
        }),
        // The Driver stores this key in snake case. The switch was of the old module.
        device.remove({ path: ["properties"], key: "switched_at" }),
      ]);
      moved = true;
    } finally {
      if (!moved) await startTasks(stopped);
    }
    await client.devices.delete(slave.key).catch((e: unknown) => {
      const then = stopped.length === 0 ? "" : " Then start your tasks again.";
      throw new Error(
        `Failed to delete ${slave.name} after the move. Delete it yourself.${then}`,
        { cause: e },
      );
    });
    return { old, stopped };
  },
});
