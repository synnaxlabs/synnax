// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { Button, Flex, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

interface ModeSelectorProps {
  onSelect: (mode: arc.Mode) => void;
}

export const ModeSelector = ({ onSelect }: ModeSelectorProps): ReactElement => {
  const handleGraphSelect = useCallback(() => onSelect("graph"), [onSelect]);
  const handleTextSelect = useCallback(() => onSelect("text"), [onSelect]);
  return (
    <Flex.Box style={{ height: "100%" }} align="center" justify="center" direction="y">
      <Flex.Box direction="y" align="center" gap="large">
        <Text.Text level="h3">Select Arc Type</Text.Text>
        <Flex.Box pack>
          <Button.Button onClick={handleGraphSelect} variant="outlined">
            <Icon.Schematic />
            Graph
          </Button.Button>
          <Button.Button onClick={handleTextSelect} variant="outlined">
            <Icon.Text />
            Text
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
