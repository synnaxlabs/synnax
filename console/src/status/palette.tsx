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
import { CREATE_LAYOUT } from "@/status/Create";
import { EXPLORER_LAYOUT } from "@/status/Explorer";

const useUpdateVisible = () => Access.useUpdateGranted(status.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(status.TYPE_ONTOLOGY_ID);

export const CreateCommand = Palette.createSimpleCommand({
  key: "create_status",
  name: "Create a Status",
  icon: <Status.CreateIcon />,
  layout: CREATE_LAYOUT,
  useVisible: useUpdateVisible,
});

export const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open_status_explorer",
  name: "Open Status Explorer",
  icon: <Icon.Explore />,
  layout: EXPLORER_LAYOUT,
  useVisible: useViewVisible,
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
