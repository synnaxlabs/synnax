// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Text } from "@synnaxlabs/pluto";

import { CONFIGURE_LAYOUTS, getIcon, getMake } from "@/hardware/device/make";
import { getKeyFromStatus } from "@/hardware/device/useListenForChanges";
import { Layout } from "@/layout";
import { type Notifications } from "@/notifications";

const notificationAdapter: Notifications.Adapter = (status) => {
  const key = getKeyFromStatus(status);
  if (key == null) return null;
  const sugared: Notifications.Sugared = { ...status };
  const make = getMake(status.data?.make);
  const startIcon = getIcon(make);
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

const ConfigureButton = ({ layout }: ConfigureButtonProps) => {
  const placeLayout = Layout.usePlacer();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => placeLayout(layout)}>
      Configure
    </Button.Button>
  );
};

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] = [notificationAdapter];
