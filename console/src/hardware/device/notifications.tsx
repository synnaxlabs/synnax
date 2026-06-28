// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Button, Icon, Text } from "@synnaxlabs/pluto";

import { CONFIGURE_MODALS, getIcon, getMake, type Make } from "@/hardware/device/make";
import { getKeyFromStatus } from "@/hardware/device/useListenForChanges";
import { Modals } from "@/layered/service/modals";
import { type Notifications } from "@/notifications";

const shouldShowConfigureButton = (make: Make): boolean =>
  make === "NI" || make === "LabJack" || make === "ethercat";

const notificationAdapter: Notifications.Adapter<ReturnType<typeof device.deviceZ>> = (
  status,
) => {
  const key = getKeyFromStatus(status);
  if (key == null) return null;
  const sugared: Notifications.Sugared = { ...status };
  const make = getMake(status.details?.make);
  const startIcon = getIcon(make);
  sugared.content = (
    <Text.Text>
      {startIcon}
      {status.message}
    </Text.Text>
  );
  if (make != null && shouldShowConfigureButton(make))
    sugared.actions = (
      <ConfigureButton modal={CONFIGURE_MODALS[make]} deviceKey={key} />
    );
  return sugared;
};

interface ConfigureButtonProps {
  modal: Modals.OpenHook<{ deviceKey?: string; title?: string }>;
  deviceKey: string;
}

const ConfigureButton = ({ modal, deviceKey }: ConfigureButtonProps) => {
  const { open } = Modals.use();
  return (
    <Button.Button
      variant="outlined"
      size="tiny"
      onClick={() => open(modal, { deviceKey })}
    >
      <Icon.Hardware />
      Configure
    </Button.Button>
  );
};

export const NOTIFICATION_ADAPTERS: Notifications.Adapter<any>[] = [
  notificationAdapter,
];
