// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Access, Channel as PChannel } from "@synnaxlabs/pluto";

import { useCalculatedModal } from "@/platform/channel/useCalculatedModal";
import { useCreateModal } from "@/platform/channel/useCreateModal";
import { Palette } from "@/platform/palette";

const useVisible = () => Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);

const CreateCommand = Palette.createCommand({
  key: "create_channel",
  name: "Create a channel",
  icon: <PChannel.CreateIcon />,
  useOnSelect: useCreateModal,
  useVisible,
});

const CreateCalculatedCommand = Palette.createCommand({
  key: "create_calculated_channel",
  name: "Create a calculated channel",
  icon: <PChannel.CreateCalculatedIcon />,
  useOnSelect: useCalculatedModal,
  useVisible,
});

export const COMMANDS = [CreateCommand, CreateCalculatedCommand];
