// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni, NotFoundError } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";
import { z } from "zod";

import { TYPED_HARDWARE } from "@/feature/ni/device/hardware";
import { Device } from "@/platform/device";
import type { Notifications } from "@/platform/notifications";

const detailsZ = z.object({ unconfiguredNIDevice: ni.device.keyZ });

/** Notifies once for each typed NI device that the Driver finds unconfigured. */
export const useListenForChanges = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const notified = useRef(new Set<ni.device.Key>());
  useEffect(
    () =>
      client?.ni.devices.onSet((dev) => {
        if (dev.identifier !== "" || notified.current.has(dev.key)) return;
        notified.current.add(dev.key);
        addStatus<typeof detailsZ>({
          variant: "info",
          name: dev.name,
          message: `${dev.model} connected`,
          details: { unconfiguredNIDevice: dev.key },
        });
      }),
    [client, addStatus],
  );
};

const Notification: Notifications.Notification = ({ status, silence }) => {
  const configure = Device.useConfigureModal();
  const { unconfiguredNIDevice: deviceKey } = detailsZ.parse(status.details);
  return (
    <Status.Notification
      status={status}
      silence={silence}
      icon={TYPED_HARDWARE.icon}
      actions={
        <Button.Button
          variant="outlined"
          size="tiny"
          onClick={() => configure({ deviceKey, hardware: TYPED_HARDWARE })}
        >
          <Icon.Hardware />
          Configure
        </Button.Button>
      }
    />
  );
};
Notification.match = (status) => detailsZ.safeParse(status.details).success;

export const NOTIFICATIONS: Notifications.Notification[] = [Notification];

/** The notification rule for typed NI device statuses. A device is in use when it has
 * an identifier. */
export const NOTIFICATION_RULE: Notifications.Rule = {
  kind: () => "state",
  inUse: async ({ client, key }) => {
    try {
      return (await client.ni.devices.retrieve({ key })).identifier !== "";
    } catch (e) {
      if (NotFoundError.matches(e)) return false;
      throw errors.fromUnknown(e);
    }
  },
  warningsDelayed: true,
};
