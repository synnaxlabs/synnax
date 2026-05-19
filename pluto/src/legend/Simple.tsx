// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Theming } from "@synnaxlabs/lyra/theming";
import { type optional } from "@synnaxlabs/x/optional";
import { type ReactElement, useState } from "react";

import { Container, type ContainerProps } from "@/legend/Container";
import { Entries, type EntryData } from "@/legend/Entries";

export interface SimpleProps extends Omit<
  ContainerProps,
  "value" | "onChange" | "background" | "draggable" | "gap"
> {
  data?: optional.Optional<EntryData, "visible">[];
  onEntryChange?: (value: EntryData) => void;
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
  allowEntryVisibleChange?: boolean;
  background?: Theming.Shade;
}

export const Simple = ({
  data = [],
  onEntryChange,
  position,
  onPositionChange,
  allowEntryVisibleChange = true,
  background = 1,
  ...rest
}: SimpleProps): ReactElement | null => {
  const [pickerVisible, setPickerVisible] = useState(false);
  if (data.length === 0) return null;
  return (
    <Container
      {...rest}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
      background={background}
      empty
    >
      <Entries
        data={data}
        onEntryChange={onEntryChange}
        colorPickerVisible={pickerVisible}
        onColorPickerVisibleChange={setPickerVisible}
        allowVisibleChange={allowEntryVisibleChange}
        background={background}
        entryProps={entryProps}
      />
    </Container>
  );
};

const entryProps = { justify: "between", grow: true } as const;
