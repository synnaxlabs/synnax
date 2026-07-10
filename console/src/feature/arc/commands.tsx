// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { EXPLORER_LAYOUT } from "@/feature/arc/Explorer";
import { Arc } from "@/platform/arc";
import { Command } from "@/platform/command";

export const CreateCommand = Command.create({
  key: "create_arc",
  name: "Create an Arc automation",
  icon: <Icon.Arc />,
  useVisible: () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID),
  useOnSelect: Arc.useCreate,
});

export const OpenExplorerCommand = Command.create({
  key: "open_arc_explorer",
  name: "Open the Arc Explorer",
  icon: <Icon.Explore />,
  useOnSelect: Command.createPlacerUseOnSelect(EXPLORER_LAYOUT),
  useVisible: () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
