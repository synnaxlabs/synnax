// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { Log } from "@/log";
import { ImportIcon } from "@/log/services/Icon";
import { import_ } from "@/log/services/import";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  key: "create-log",
  name: "Create Log",
  icon: <Icon.Log />,
  onSelect: ({ placeLayout }) => placeLayout(Log.create({})),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-log",
  name: "Import Log(s)",
  icon: <ImportIcon />,
  onSelect: import_,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
