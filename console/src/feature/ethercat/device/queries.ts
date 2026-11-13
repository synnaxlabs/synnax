// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, status } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";
import { array, primitive, TimeSpan, unique, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { SLAVE_SCHEMAS, type SlaveDevice } from "@/feature/ethercat/device/types";
import {
  type Channel,
  READ_TYPE,
  SCAN_TYPE,
  WRITE_TYPE,
} from "@/feature/ethercat/task/types";

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

export interface SetStandaloneParams {
  slave: SlaveDevice;
}

/**
 * How long a switch waits for the Driver. Writes to several slaves can be slow. It ends
 * before the default 60 s operation timeout of the Driver, whose error reply would read
 * as a refusal while the switch still runs.
 */
const SWITCH_TIMEOUT = TimeSpan.seconds(50);

/**
 * Stops each EtherCAT task that runs on the network of the slave, then switches the
 * slave to Synnax only. The slave changes after its power restarts. When a stop fails
 * or the Driver refuses the switch, starts the tasks again. When the Driver gives no
 * answer, the switch may still run, so the tasks stay stopped.
 */
export const { useUpdate: useSetStandalone } = Flux.createUpdate<SetStandaloneParams>({
  name: "EtherCAT slave",
  verbs: SWITCH,
  update: async ({ client, data }) => {
    const { rack, key, properties } = data.slave;
    const running = (await client.tasks.retrieve({ rack, includeStatus: true })).filter(
      (t) =>
        (t.type === READ_TYPE || t.type === WRITE_TYPE) &&
        t.status?.details.running === true,
    );
    const devicesOf = (config: unknown): string[] =>
      channelDevicesZ.parse(config).channels.map((ch) => ch.device);
    const keys = unique.unique(running.flatMap((t) => devicesOf(t.config)));
    const onNetwork = new Set(
      (keys.length === 0
        ? []
        : await client.devices.retrieve({ key: keys, schemas: SLAVE_SCHEMAS })
      )
        .filter((s) => s.properties.network === properties.network)
        .map((s) => s.key),
    );
    const tasks = running.filter((t) =>
      devicesOf(t.config).some((k) => onNetwork.has(k)),
    );
    const scanTask = await client.tasks.retrieve({ type: SCAN_TYPE, rack, one: true });
    const restart = async (): Promise<void> => {
      // A task that fails to start shows its own error.
      await Promise.allSettled(
        tasks.map(async (t) => await t.executeCommandSync({ type: "start" })),
      );
    };
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
      if (!stopped) await restart();
    }
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
      await restart();
      throw status.toError(res);
    }
    return data;
  },
});
