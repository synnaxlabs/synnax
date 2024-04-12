// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type device } from "@synnaxlabs/client";
import { Button, Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { type change } from "@synnaxlabs/x";

import { create } from "@/hardware/ni/new/Configure";
import { Layout } from "@/layout";
import { type NotificationAdapter } from "@/palette/Notifications";

export const useListenForChanges = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();

  useAsyncEffect(async () => {
    if (client == null) return;
    const tracker = await client.hardware.devices.openDeviceTracker();
    tracker.onChange((changes) => {
      const sets = changes.filter(({ variant }) => variant === "set") as Array<
        change.Set<string, device.Device>
      >;
      sets.forEach(({ value: dev }) => {
        addStatus({
          key: `new-device-${dev.key}`,
          variant: "info",
          message: `New ${dev.model} connected`,
          data: dev,
        });
      });
    });
    return () => {
      void tracker.close();
    };
  }, [client, addStatus]);
};

export const notificationAdapter: NotificationAdapter = (status) => {
  if (!status.key.startsWith("new-device-")) return null;
  // grab the device key from the status key
  const deviceKey = status.key.slice("new-device-".length);
  return {
    ...status,
    actions: [<ConfigureButton deviceKey={deviceKey} key="configure" />],
  };
};

interface ConfigureButtonProps {
  deviceKey: string;
}

const ConfigureButton = ({ deviceKey }: ConfigureButtonProps): ReactElement => {
  const place = Layout.usePlacer();

  return (
    <Button.Button
      variant="outlined"
      size="small"
      onClick={() => place(create(deviceKey, {}))}
    >
      Configure
    </Button.Button>
  );
};
