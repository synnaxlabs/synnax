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

import { Channel } from "@/channel";
import { Palette } from "@/palette";

const useVisible = () => Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);

const CreateCommand = Palette.createSimpleCommand({
  key: "create-channel",
  name: "Create a channel",
  icon: <PChannel.CreateIcon />,
  layout: Channel.CREATE_LAYOUT,
  useVisible,
});

const CreateCalculatedCommand = Palette.createSimpleCommand({
  key: "create-calculated-channel",
  name: "Create a calculated channel",
  icon: <PChannel.CreateCalculatedIcon />,
  layout: Channel.CALCULATED_LAYOUT,
  useVisible,
});

export const COMMANDS = [CreateCommand, CreateCalculatedCommand];
