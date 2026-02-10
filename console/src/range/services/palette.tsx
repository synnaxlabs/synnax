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

import { Palette } from "@/palette";
import { Range } from "@/range";

const useUpdateVisible = () => Access.useUpdateGranted(ranger.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID);

export const CreateCommand = Palette.createSimpleCommand({
  key: "define-range",
  name: "Create a range",
  icon: <Ranger.CreateIcon />,
  layout: Range.CREATE_LAYOUT,
  useVisible: useUpdateVisible,
});

export const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open-explorer",
  name: "Open the Range Explorer",
  icon: <Icon.Explore />,
  layout: Range.EXPLORER_LAYOUT,
  useVisible: useViewVisible,
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
