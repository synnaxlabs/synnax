// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { useConnectModal } from "@/feature/device/Connect";
import { HARDWARE } from "@/feature/device/integrations";
import { Command } from "@/platform/command";
import { Rack } from "@/platform/rack";

const useVisible = () => Access.useCreateGranted(device.TYPE_ONTOLOGY_ID);

const ConnectCommand = Command.create({
  key: "device_connect",
  name: "Connect device",
  icon: <Icon.Device />,
  useOnSelect: useConnectModal,
  useVisible,
});

const createUseConnect = (integration: string) => {
  const useConnect = () => {
    const openConnect = useConnectModal();
    return () => openConnect({ integration });
  };
  return useConnect;
};

const HARDWARE_COMMANDS = HARDWARE.map(({ key, title, icon }) =>
  Command.create({
    key: `device_connect_${key}`,
    name: `Connect ${title}`,
    icon,
    useOnSelect: createUseConnect(key),
    useVisible,
  }),
);

export const CONNECT_COMMANDS: Command.Command[] = [
  ConnectCommand,
  ...HARDWARE_COMMANDS,
];

export const CREATE_RACK_COMMAND = Command.create({
  key: "rack_create",
  name: "Create rack",
  icon: <Icon.Rack />,
  useOnSelect: Rack.useCreateModal,
  useVisible: () => Access.useCreateGranted(rack.TYPE_ONTOLOGY_ID),
});
