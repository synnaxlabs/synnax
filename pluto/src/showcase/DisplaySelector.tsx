// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";

import { Button } from "@/button";
import { Flex } from "@/flex";
import { Select } from "@/select";
import { type Display, DISPLAYS } from "@/showcase/constants";
import { Text } from "@/text";

interface DisplaySelectorProps {
  displays: Display[];
  setDisplays: (displays: Display[]) => void;
}

export const DisplaySelector = ({ displays, setDisplays }: DisplaySelectorProps) => {
  const handleSelectAll = () => setDisplays([...DISPLAYS]);
  const handleClearAll = () => setDisplays([]);
  return (
    <Flex.Box y gap="small">
      <Flex.Box x gap="small" align="center">
        <Text.Text level="p" weight={500}>
          Select Components:
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          onClick={handleSelectAll}
          disabled={displays.length === DISPLAYS.length}
        >
          Select All
        </Button.Button>
        <Button.Button
          variant="text"
          size="small"
          onClick={handleClearAll}
          disabled={displays.length === 0}
        >
          Clear All
        </Button.Button>
        <Text.Text level="small" style={{ opacity: 0.6 }}>
          {displays.length} of {DISPLAYS.length} selected
        </Text.Text>
      </Flex.Box>
      <Select.Buttons multiple keys={DISPLAYS} value={displays} onChange={setDisplays}>
        {DISPLAYS.map((d) => (
          <Select.Button key={d} itemKey={d}>
            {caseconv.capitalize(d)}
          </Select.Button>
        ))}
      </Select.Buttons>
    </Flex.Box>
  );
};
