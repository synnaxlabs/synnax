// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, rack, status, task } from "@synnaxlabs/client";
import { destructor, errors, primitive } from "@synnaxlabs/x";

import { type Device } from "@/platform/device";
import { type Notifications } from "@/platform/notifications";

/** The make and scan task type of an integration. */
export interface Scanner extends Pick<Device.Integration, "make" | "scanType"> {}

const commandDetailsZ = task.statusDetailsZ().pick({ cmd: true });
const rackDetailsZ = task.statusDetailsZ().pick({ rack: true });

const retrieveTask = async ({
  client,
  key,
}: Notifications.InUseParams): Promise<task.Task | null> => {
  try {
    return await client.tasks.retrieve({ key });
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

/**
 * Creates the notification rule for task statuses. A status that answers a command is a
 * reply. A scanner, which is an internal task whose type one of the scanners names, is
 * in use when its rack has a configured device of the scanner's make. Every other task
 * is always in use. The open problem of a task's rack covers the task's problems.
 * @param scanners - The scanner of each integration.
 */
export const createNotificationRule = (scanners: Scanner[]): Notifications.Rule => ({
  kind: (stat) =>
    primitive.isNonZero(commandDetailsZ.safeParse(status.detailsOf(stat)).data?.cmd)
      ? "reply"
      : "state",
  inUse: async (args) => {
    const tsk = await retrieveTask(args);
    if (tsk == null) return false;
    if (!tsk.internal) return true;
    const make = scanners.find(({ scanType }) => scanType === tsk.type)?.make;
    if (make == null) return true;
    const devices = await args.client.devices.retrieve({
      racks: [tsk.rack],
      makes: [make],
    });
    return devices.some(({ configured }) => configured);
  },
  onUseChange: (client, handler) => {
    const removeSetListener = client.devices.onSet(() => handler());
    const removeDeleteListener = client.devices.onDelete(() => handler());
    return () => destructor.unwind(removeSetListener, removeDeleteListener);
  },
  warningsDelayed: true,
  coveredBy: (stat) => {
    const r = rackDetailsZ.safeParse(status.detailsOf(stat)).data?.rack;
    return primitive.isZero(r) ? null : rack.statusKey(r);
  },
});
