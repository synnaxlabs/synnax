// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type Optional } from "@synnaxlabs/x";
import { memo, type ReactElement, useState } from "react";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type state } from "@/state";
import { Text } from "@/text";
import { type Theming } from "@/theming";
import { Container, type ContainerProps } from "@/vis/legend/Container";

interface SimpleEntry {
  key: string;
  label: string;
  color: color.Crude;
  visible: boolean;
  axis?: string;
}

export interface SimpleProps
  extends Omit<ContainerProps, "value" | "onChange" | "background"> {
  data?: Optional<SimpleEntry, "visible">[];
  onEntryChange?: (value: SimpleEntry) => void;
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
  allowVisibleChange?: boolean;
  background?: Theming.Shade;
}

interface LegendSwatchesProps
  extends Pick<SimpleProps, "onEntryChange" | "background"> {
  data: Optional<SimpleEntry, "visible">[];
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
        .map(({ key, color, label, visible = true, axis }) => (
          <Flex.Box
            key={key}
            className={CSS.B("legend-swatch")}
            x
            align="center"
            gap="small"
            justify="between"
          >
            <Flex.Box direction="x" align="center" gap="small">
              <Color.Swatch
                value={color}
                onChange={(c) =>
                  onEntryChange?.({ key, color: c, label, visible, axis })
                }
                onVisibleChange={onVisibleChange}
                allowChange={onEntryChange != null}
                draggable={false}
                size="tiny"
              />
              {axis != null && (
                <Text.Text level="small" color={8}>
                  {axis.toUpperCase()}
                </Text.Text>
              )}
              <Text.MaybeEditable
                level="small"
                value={label}
                onChange={(l) =>
                  onEntryChange?.({ key, color, label: l, visible, axis })
                }
                overflow="nowrap"
                color={visible ? 10 : 7}
                onDoubleClick={(e) => e.stopPropagation()}
              />
            </Flex.Box>
            {allowVisibleChange && (
              <Button.Button
                className={CSS.B("visible-toggle")}
                onClick={(e) => {
                  e.stopPropagation();
                  onEntryChange?.({ key, color, label, visible: !visible, axis });
                }}
                onDoubleClick={(e) => e.stopPropagation()}
                size="tiny"
                contrast={background}
                variant="text"
              >
                {visible ? <Icon.Visible /> : <Icon.Hidden />}
              </Button.Button>
            )}
          </Flex.Box>
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
