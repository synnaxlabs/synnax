// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Log } from "@/log";
import { CreateIcon, ImportIcon } from "@/log/services/Icon";
import { import_ } from "@/log/services/import";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  key: "create-log",
  name: "Create a Log",
  icon: <CreateIcon />,
  onSelect: ({ placeLayout }) => placeLayout(Log.create()),
  visible: ({ store, client }) =>
    Access.editGranted({ id: log.TYPE_ONTOLOGY_ID, store, client }),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-log",
  name: "Import Log(s)",
  sortOrder: -1,
  icon: <ImportIcon />,
  onSelect: import_,
  visible: ({ store, client }) =>
    Access.editGranted({ id: log.TYPE_ONTOLOGY_ID, store, client }),
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
