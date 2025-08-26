// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { type Theming } from "@/theming";
import { Container, type ContainerProps } from "@/vis/legend/Container";
import { Entries } from "@/vis/legend/Entries";
import { type EntryData } from "@/vis/legend/Entry";

export interface SimpleProps
  extends Omit<ContainerProps, "value" | "onChange" | "background"> {
  data?: Optional<EntryData, "visible">[];
  onEntryChange?: (value: EntryData) => void;
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
  allowVisibleChange?: boolean;
  background?: Theming.Shade;
}

export const Simple = ({
  data = [],
  onEntryChange,
  position,
  onPositionChange,
  allowVisibleChange = true,
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
      gap={allowVisibleChange ? 0 : "small"}
      background={background}
    >
      <Entries
        data={data}
        onEntryChange={onEntryChange}
        onVisibleChange={setPickerVisible}
        allowVisibleChange={allowVisibleChange}
        background={background}
      />
    </Container>
  );
};
