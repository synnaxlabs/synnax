// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { LAYOUT } from "@/hardware/task/sequence/Sequence";
import { type Palette } from "@/palette";

const CREATE_SEQUENCE_COMMAND: Palette.Command = {
  key: "create-sequence",
  name: "Create Sequence",
  icon: <Icon.Control />,
  onSelect: ({ placeLayout, handleException, rename }) => {
    rename({}, { icon: "Control", name: "Control.Sequence.Create" })
      .then((result) => {
        if (result == null) return null;
        return placeLayout({ ...LAYOUT, name: result });
      })
      .catch(handleException);
  },
};

export const COMMANDS = [CREATE_SEQUENCE_COMMAND];
