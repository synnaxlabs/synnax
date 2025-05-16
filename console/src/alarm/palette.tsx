// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { ALARM_LAYOUT } from "@/alarm/Alarm";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  icon: <Icon.Channel />,
  name: "Create Alarm",
  key: "create-alarm",
  onSelect: ({ placeLayout }) => placeLayout(ALARM_LAYOUT),
  endContent: [],
};

export const COMMANDS = [CREATE_COMMAND];
