// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type Optional } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Text } from "@/text";
import { Container, type ContainerProps } from "@/vis/legend/Container";

interface SimpleEntry {
  key: string;
  label: string;
  color: Color.Crude;
  visible: boolean;
}

export interface SimpleProps extends Omit<ContainerProps, "value" | "onChange"> {
  data?: Optional<SimpleEntry, "visible">[];
  onEntryChange?: (value: SimpleEntry) => void;
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
  allowVisibleChange?: boolean;
}

export const legendSwatches = (
  data: Optional<SimpleEntry, "visible">[],
  onEntryChange: SimpleProps["onEntryChange"],
  onVisibleChange?: (visible: boolean) => void,
  allowVisibleChange = true,
): ReactElement[] =>
  data
    .sort((a, b) => a.label.localeCompare(b.label))
    .map(({ key, color, label, visible = true }) => (
      <Align.Space
        key={key}
        style={{ cursor: "pointer" }}
        x
        align="center"
        size="small"
        justify="spaceBetween"
        grow
      >
        <Align.Space direction="x" align="center" size="small">
          <Color.Swatch
            value={color}
            onChange={(c) => onEntryChange?.({ key, color: c, label, visible })}
            onVisibleChange={onVisibleChange}
            allowChange={onEntryChange != null}
            draggable={false}
            size="tiny"
          />
          <Text.MaybeEditable
            level="small"
            value={label}
            onChange={(l) => onEntryChange?.({ key, color, label: l, visible })}
            noWrap
            shade={visible ? 10 : 7}
            onDoubleClick={(e) => e.stopPropagation()}
          />
        </Align.Space>
        {allowVisibleChange && (
          <Button.Icon
            className={CSS.B("visible-toggle")}
            onClick={(e) => {
              e.stopPropagation();
              onEntryChange?.({ key, color, label, visible: !visible });
            }}
            onDoubleClick={(e) => e.stopPropagation()}
            size="tiny"
            shade={2}
          >
            {visible ? <Icon.Visible /> : <Icon.Hidden />}
          </Button.Icon>
        )}
      </Align.Space>
    ));

export const Simple = ({
  data = [],
  onEntryChange,
  position,
  onPositionChange,
  allowVisibleChange = true,
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
      empty
    >
      {legendSwatches(data, onEntryChange, setPickerVisible, allowVisibleChange)}
    </Container>
  );
};
