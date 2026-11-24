// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Channel as PChannel } from "@synnaxlabs/pluto";

import { Explorer } from "@/feature/channel/explorer";
import { Channel as PlatformChannel } from "@/platform/channel";
import { Command } from "@/platform/command";

const useVisible = () => Access.useCreateGranted(channel.TYPE_RESOURCE_ID);

const CreateCommand = Command.create({
  key: "create_channel",
  name: "Create channel",
  icon: <PChannel.CreateIcon />,
  useOnSelect: PlatformChannel.useCreateModal,
  useVisible,
});

const CreateCalculatedCommand = Command.create({
  key: "create_calculated_channel",
  name: "Create calculated channel",
  icon: <PChannel.CreateCalculatedIcon />,
  useOnSelect: PlatformChannel.useCalculatedModal,
  useVisible,
});

const OpenExplorerCommand = Command.create({
  key: "open_channel_explorer",
  name: "Open channel explorer",
  icon: <Icon.Channel />,
  useOnSelect: Explorer.useOpenTab,
  useVisible: () => Access.useRetrieveGranted(channel.TYPE_RESOURCE_ID),
});

export const COMMANDS = [CreateCommand, CreateCalculatedCommand, OpenExplorerCommand];
