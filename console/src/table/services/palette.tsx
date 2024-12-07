// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { type Command } from "@/palette/Palette";
import { create } from "@/table/Table";

const createCommand: Command = {
  key: "create-table",
  name: "Create a Table",
  icon: <Icon.Table />,
  onSelect: ({ placeLayout }) => placeLayout(create({})),
};

export const COMMANDS = [createCommand];
