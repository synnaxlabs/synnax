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

import { Arc } from "@/platform/arc";
import { Palette } from "@/platform/palette";

import { Explorer } from "./explorer";

const useCreateVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID);

export const CreateCommand = Palette.createCommand({
  key: "create_arc",
  name: "Create an Arc Automation",
  icon: <Icon.Arc />,
  useOnSelect: Arc.useCreate,
  useVisible: useCreateVisible,
});

export const OpenExplorerCommand = Palette.createCommand({
  key: "open_arc_explorer",
  name: "Open the Arc Explorer",
  icon: <Icon.Explore />,
  useOnSelect: Explorer.useOpenTab,
  useVisible: useViewVisible,
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
