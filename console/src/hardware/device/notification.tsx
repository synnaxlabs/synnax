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

import { Device } from "@/hardware/device";
import { CONFIGURE_LAYOUTS, getMake, MAKE_ICONS } from "@/hardware/device/make";
import { Layout } from "@/layout";
import { type Notifications } from "@/notifications";

export const notificationAdapter: Notifications.Adapter = (status) => {
  const key = Device.getKeyFromStatus(status);
  if (key == null) return null;
  const sugared: Notifications.Sugared = { ...status };
  const make = getMake(status.data?.make);
  const startIcon = make ? MAKE_ICONS[make] : <Icon.Device />;
  sugared.content = (
    <Text.WithIcon level="p" startIcon={startIcon}>
      {status.message}
    </Text.WithIcon>
  );
  if (make)
    sugared.actions = <ConfigureButton layout={{ ...CONFIGURE_LAYOUTS[make], key }} />;
  return sugared;
};

interface ConfigureButtonProps {
  layout: Layout.BaseState;
}

const ConfigureButton = ({ layout }: ConfigureButtonProps): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => placeLayout(layout)}>
      Configure
    </Button.Button>
  );
};
