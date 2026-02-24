// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { Spectrogram } from "@/spectrogram";

export const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const handleSelect = useCallback(
    () => placeLayout(Spectrogram.create()),
    [placeLayout],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a spectrogram"
      icon={<Icon.Visualize />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-spectrogram";
CreateCommand.commandName = "Create a spectrogram";

export const COMMANDS = [CreateCommand];
