// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Select } from "@synnaxlabs/lyra/select";
import type { bounds } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { RangeInput } from "@/schematic/node/common/form/Range";
import { Telem } from "@/schematic/node/common/telem";

export interface MatchInputProps {
  value: schematic.symbol.Matcher;
  onChange: (next: schematic.symbol.Matcher) => void;
  /** The range that a switch to a within match starts with. */
  within?: bounds.Bounds;
}

/**
 * Picks how a state reads a value: as it is, or as 1 inside a range and 0 outside it.
 * A within match shows its range beside the picker, or under it when the line is full.
 */
export const MatchInput = ({
  value,
  onChange,
  within = Telem.DEFAULT_THRESHOLD,
}: MatchInputProps): ReactElement => (
  <Flex.Box x align="center" gap="medium" wrap>
    <Select.Buttons
      value={value.type}
      onChange={(type: schematic.symbol.MatcherType) =>
        onChange(type === "exact" ? { type } : { type, bounds: within })
      }
    >
      <Select.Item itemKey="exact" tooltip="Reads each value as it is">
        Exact
      </Select.Item>
      <Select.Item
        itemKey="within"
        tooltip="Reads a value in the range as 1, and any other value as 0"
      >
        Within
      </Select.Item>
    </Select.Buttons>
    {value.type === "within" && (
      <RangeInput
        label="Match"
        value={value.bounds}
        onChange={(b) => onChange({ ...value, bounds: b })}
      />
    )}
  </Flex.Box>
);
