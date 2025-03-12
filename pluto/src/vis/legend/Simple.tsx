// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { Align } from "@/align";
import { Color } from "@/color";
import { Text } from "@/text";
import { Container, type ContainerProps } from "@/vis/legend/Container";

interface SimpleEntry {
  key: string;
  label: string;
  color: Color.Crude;
}

export interface SimpleProps extends Omit<ContainerProps, "value" | "onChange"> {
  data?: SimpleEntry[];
  onEntryChange?: (value: SimpleEntry) => void;
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
}

export const legendSwatches = (
  data: SimpleEntry[],
  onEntryChange: SimpleProps["onEntryChange"],
  onVisibleChange?: (visible: boolean) => void,
): ReactElement[] =>
  data
    .sort((a, b) => a.label.localeCompare(b.label))
    .map(({ key, color, label }) => (
      <Align.Space
        key={key}
        style={{ cursor: "pointer" }}
        direction="x"
        align="center"
        size="small"
      >
        <Color.Swatch
          value={color}
          onChange={(c) => onEntryChange?.({ key, color: c, label })}
          onVisibleChange={onVisibleChange}
          allowChange={onEntryChange != null}
          draggable={false}
          size="tiny"
        />
        <Text.MaybeEditable
          level="small"
          value={label}
          onChange={(l) => onEntryChange?.({ key, color, label: l })}
          noWrap
        />
      </Align.Space>
    ));

export const Simple = ({
  data = [],
  onEntryChange,
  position,
  onPositionChange,
  ...rest
}: SimpleProps): ReactElement | null => {
  const [pickerVisible, setPickerVisible] = useState<boolean>(false);

  if (data.length === 0) return null;

  return (
    <Container
      {...rest}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
    >
      {legendSwatches(data, onEntryChange, setPickerVisible)}
    </Container>
  );
};
