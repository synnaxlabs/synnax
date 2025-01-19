// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Button, type Icon as PIcon, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Device } from "@/hardware/device";
import { CONFIGURE_LAYOUTS } from "@/hardware/device/services/configureLayouts";
import { type Make, makeZ } from "@/hardware/device/services/make";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { type Notifications } from "@/notifications";

const MAKE_ICONS: Record<Make, PIcon.Element> = {
  [LabJack.Device.MAKE]: <Icon.Logo.LabJack />,
  [NI.Device.MAKE]: <Icon.Logo.NI />,
  [OPC.Device.MAKE]: <Icon.Logo.OPC />,
};

export const notificationAdapter: Notifications.NotificationAdapter = (status) => {
  const key = Device.getKeyFromStatus(status);
  if (key == null) return null;
  const sugared: Notifications.SugaredNotification = { ...status };
  const make = makeZ.safeParse(status?.data?.make)?.data;
  const startIcon = make != null ? MAKE_ICONS[make] : <Icon.Device />;
  sugared.content = (
    <Text.WithIcon level="p" startIcon={startIcon}>
      {status.message}
    </Text.WithIcon>
  );
  if (make != null)
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
