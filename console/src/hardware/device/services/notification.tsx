// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Button, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type Make, MAKE_ICONS } from "@/hardware/makes";
import { createConfigureLayout } from "@/hardware/ni/device/Configure";
import { Layout } from "@/layout";
import {
  type NotificationAdapter,
  type SugaredNotification,
} from "@/notifications/Notifications";

export const notificationAdapter: NotificationAdapter = (status) => {
  if (!status.key.startsWith("new-device-")) return null;
  // grab the device key from the status key
  const deviceKey = status.key.slice("new-device-".length);
  const sugared: SugaredNotification = {
    ...status,
    actions: [<ConfigureButton deviceKey={deviceKey} key="configure" />],
  };
  const icon = MAKE_ICONS[status?.data?.make as Make] ?? <Icon.Device />;
  sugared.content = (
    <Text.WithIcon level="p" startIcon={icon}>
      {status.message}
    </Text.WithIcon>
  );
  return sugared;
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
      onClick={() => place(createConfigureLayout(deviceKey, {}))}
    >
      Configure
    </Button.Button>
  );
};
