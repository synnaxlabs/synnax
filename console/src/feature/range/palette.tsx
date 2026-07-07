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

import { Explorer } from "@/feature/range/explorer";
import { Palette } from "@/platform/palette";
import { Range } from "@/platform/range";

export const CreateCommand = Palette.createCommand({
  key: "define_range",
  name: "Create a range",
  icon: <Ranger.CreateIcon />,
  useOnSelect: Range.useCreateModal,
  useVisible: () => Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID),
});

export const OpenExplorerCommand = Palette.createCommand({
  key: "open_explorer",
  name: "Open the Range Explorer",
  icon: <Icon.Explore />,
  useOnSelect: Explorer.useOpenTab,
  useVisible: () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
