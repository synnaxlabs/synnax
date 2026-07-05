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

import { Palette } from "@/palette";
import { EXPLORER_LAYOUT } from "@/status/Explorer";
import { useCreateModal } from "@/status/useCreateModal";

const CreateCommand = Palette.createCommand({
  key: "create_status",
  name: "Create a status",
  icon: <Status.CreateIcon />,
  useOnSelect: useCreateModal,
  useVisible: () => Access.useCreateGranted(status.TYPE_ONTOLOGY_ID),
});

const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open_status_explorer",
  name: "Open the Status Explorer",
  icon: <Icon.Explore />,
  layout: EXPLORER_LAYOUT,
  useVisible: () => Access.useRetrieveGranted(status.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
