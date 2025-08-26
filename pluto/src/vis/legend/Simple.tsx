// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { memo, type ReactElement, useState } from "react";

import { CSS } from "@/css";
import { type state } from "@/state";
import { type Theming } from "@/theming";
import { Container, type ContainerProps } from "@/vis/legend/Container";
import { Entry, type EntryData } from "@/vis/legend/Entry";

export interface SimpleProps
  extends Omit<ContainerProps, "value" | "onChange" | "background"> {
  data?: Optional<EntryData, "visible">[];
  onEntryChange?: (value: EntryData) => void;
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
  allowVisibleChange?: boolean;
  background?: Theming.Shade;
}

interface LegendSwatchesProps
  extends Pick<SimpleProps, "onEntryChange" | "background"> {
  data: Optional<EntryData, "visible">[];
  onEntryChange: SimpleProps["onEntryChange"];
  onVisibleChange?: state.Setter<boolean>;
  allowVisibleChange?: boolean;
}

export const LegendSwatches = memo(
  ({
    data,
    onEntryChange,
    onVisibleChange,
    allowVisibleChange = true,
    background = 1,
  }: LegendSwatchesProps): ReactElement => (
    <>
      {data
        .sort((a, b) => a.label.localeCompare(b.label))
        .map(({ key, color, label, visible = true }) => (
          <Entry
            key={key}
            entry={{ key, color, label, visible }}
            onEntryChange={onEntryChange}
            onVisibleChange={onVisibleChange}
            allowVisibleChange={allowVisibleChange}
            background={background}
          />
        ))}
    </>
  ),
);

LegendSwatches.displayName = "LegendSwatches";

export const Simple = ({
  data = [],
  onEntryChange,
  position,
  onPositionChange,
  allowVisibleChange = true,
  background = 1,
  ...rest
}: SimpleProps): ReactElement | null => {
  const [pickerVisible, setPickerVisible] = useState<boolean>(false);
  if (data.length === 0) return null;
  return (
    <Container
      {...rest}
      className={allowVisibleChange ? CSS.M("with-visible-toggle") : undefined}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
      gap={allowVisibleChange ? 0 : "small"}
      background={background}
    >
      <LegendSwatches
        data={data}
        onEntryChange={onEntryChange}
        onVisibleChange={setPickerVisible}
        allowVisibleChange={allowVisibleChange}
        background={background}
      />
    </Container>
  );
};
