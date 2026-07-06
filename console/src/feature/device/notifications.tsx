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
import { z } from "zod";

import { getIcon, getMake, type Make, useConfigureModal } from "@/feature/device/make";
import { getKeyFromStatus } from "@/feature/device/useListenForChanges";
import { type Notifications } from "@/platform/notifications";

const shouldShowConfigureButton = (make: Make): boolean =>
  make === "NI" || make === "LabJack" || make === "ethercat";

const detailsZ = z.object({ make: z.unknown() });

const notificationAdapter: Notifications.Adapter = (status) => {
  const key = getKeyFromStatus(status);
  if (key == null) return null;
  const sugared: Notifications.Sugared = { ...status };
  const make = getMake(detailsZ.safeParse(status.details).data?.make);
  const startIcon = getIcon(make);
  sugared.content = (
    <Text.Text>
      {startIcon}
      {status.message}
    </Text.Text>
  );
  if (make != null && shouldShowConfigureButton(make))
    sugared.actions = <ConfigureButton make={make} deviceKey={key} />;
  return sugared;
};

interface ConfigureButtonProps {
  make: Make;
  deviceKey: device.Key;
}

const ConfigureButton = ({ make, deviceKey }: ConfigureButtonProps) => {
  const configure = useConfigureModal();
  return (
    <Button.Button
      variant="outlined"
      size="tiny"
      onClick={() => configure(make, deviceKey)}
    >
      <Icon.Hardware />
      Configure
    </Button.Button>
  );
};

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] = [notificationAdapter];
