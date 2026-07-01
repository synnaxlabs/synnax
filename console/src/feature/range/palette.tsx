// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Access, Icon, Ranger } from "@synnaxlabs/pluto";

import { EXPLORER_LAYOUT } from "@/feature/range/Explorer";
import { Palette } from "@/primitive/palette";
import { Range } from "@/primitive/range";

export const CreateCommand = Palette.createCommand({
  key: "define_range",
  name: "Create a range",
  icon: <Ranger.CreateIcon />,
  useOnSelect: Range.useCreateModal,
  useVisible: () => Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID),
});

export const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open_explorer",
  name: "Open the Range Explorer",
  icon: <Icon.Explore />,
  layout: EXPLORER_LAYOUT,
  useVisible: () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
