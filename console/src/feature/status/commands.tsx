// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Icon, Status } from "@synnaxlabs/pluto";

import { EXPLORER_LAYOUT } from "@/feature/status/Explorer";
import { Command } from "@/platform/command";
import { Status as PlatformStatus } from "@/platform/status";

const CreateCommand = Command.create({
  key: "create_status",
  name: "Create a status",
  icon: <Status.CreateIcon />,
  useOnSelect: PlatformStatus.useCreateModal,
  useVisible: () => Access.useCreateGranted(status.TYPE_ONTOLOGY_ID),
});

const OpenExplorerCommand = Command.create({
  key: "open_status_explorer",
  name: "Open the Status Explorer",
  icon: <Icon.Explore />,
  useOnSelect: Command.createPlacerUseOnSelect(EXPLORER_LAYOUT),
  useVisible: () => Access.useRetrieveGranted(status.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
