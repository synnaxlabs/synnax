// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { errors } from "@synnaxlabs/x";
import { z } from "zod";

import { getIcon, getMake, type Make, useConfigureModal } from "@/feature/device/make";
import { getKeyFromStatus } from "@/feature/device/useListenForChanges";
import { type Notifications } from "@/platform/notifications";

const shouldShowConfigureButton = (make: Make): boolean =>
  make === "NI" || make === "LabJack" || make === "ethercat";

const detailsZ = z.object({ make: z.unknown() });

const Notification: Notifications.Notification = ({ status, silence }) => {
  const configure = useConfigureModal();
  const key = getKeyFromStatus(status);
  const make = getMake(detailsZ.safeParse(status.details).data?.make);
  const showConfigure = make != null && key != null && shouldShowConfigureButton(make);
  return (
    <Status.Notification
      status={status}
      silence={silence}
      icon={getIcon(make)}
      actions={
        showConfigure ? (
          <Button.Button
            variant="outlined"
            size="tiny"
            onClick={() => configure(make, key)}
          >
            <Icon.Hardware />
            Configure
          </Button.Button>
        ) : undefined
      }
    />
  );
};
Notification.match = (status) => getKeyFromStatus(status) != null;

export const NOTIFICATIONS: Notifications.Notification[] = [Notification];

/** The notification rule for device statuses. A device is in use when configured. */
export const NOTIFICATION_RULE: Notifications.Rule = {
  kind: () => "state",
  inUse: async ({ client, key }) => {
    try {
      return (await client.devices.retrieve({ key })).configured;
    } catch (e) {
      if (NotFoundError.matches(e)) return false;
      throw errors.fromUnknown(e);
    }
  },
  warningsDelayed: true,
};
