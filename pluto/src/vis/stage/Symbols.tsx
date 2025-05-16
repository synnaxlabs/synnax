// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/stage/Symbols.css";

import { Icon } from "@synnaxlabs/media";
import {
  type direction,
  type location,
  type UnknownRecord,
  type xy,
} from "@synnaxlabs/x";
import { type CSSProperties, type FC, type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type Control } from "@/telem/control";
import { Text } from "@/text";
import { Button as CoreButton } from "@/vis/button";
import { Primitives } from "@/vis/stage/primitives";

export interface ControlStateProps extends Omit<Align.SpaceProps, "direction"> {
  show?: boolean;
  showChip?: boolean;
  showIndicator?: boolean;
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
  orientation?: location.Outer;
}

export interface LabelExtensionProps {
  label?: string;
  level?: Text.Level;
  orientation?: location.Outer;
  direction?: direction.Direction;
  maxInlineSize?: number;
  align?: Align.Alignment;
}

export type SymbolProps<P extends object = UnknownRecord> = P & {
  symbolKey: string;
  position: xy.XY;
  aetherKey: string;
  selected: boolean;
  draggable: boolean;
  onChange: (value: Partial<P>) => void;
};

export interface ButtonProps
  extends Omit<Primitives.ButtonProps, "label" | "onClick">,
    Omit<CoreButton.UseProps, "aetherKey"> {
  label?: LabelExtensionProps;
  control?: ControlStateProps;
}

export const Button = ({
  symbolKey,
  label,
  orientation = "left",
  sink,
  control,
  selected,
  draggable,
  onChange,
  ...rest
}: SymbolProps<ButtonProps>) => {
  const { click } = CoreButton.use({ aetherKey: symbolKey, sink });
  return (
    <Primitives.Button {...label} onClick={click} orientation={orientation} {...rest} />
  );
};

export const ButtonPreview = ({ label: _, ...rest }: ButtonProps): ReactElement => (
  <Primitives.Button label="Button" {...rest} />
);

export interface SourceProps {}

export const Source = ({}: SymbolProps<SourceProps>) => (
  <Align.Space
    background={3}
    borderShade={4}
    bordered
    rounded={1}
    style={{ padding: "1rem 2rem" }}
  >
    <Text.WithIcon startIcon={<Icon.Channel />} level="p">
      Telemetry Source
    </Text.WithIcon>
  </Align.Space>
);

export interface IfStatementProps
  extends Omit<Primitives.ButtonProps, "label" | "onClick"> {
  label?: LabelExtensionProps;
}

export const CaseStatement = () => (
  <Align.Space
    background={3}
    borderShade={4}
    bordered
    rounded={1}
    style={{ padding: "1rem 2rem" }}
  >
    <Text.WithIcon startIcon={<Icon.Channel />} level="p">
      Case
    </Text.WithIcon>
  </Align.Space>
);

export interface SendNotificationProps
  extends Omit<Primitives.ButtonProps, "label" | "onClick"> {
  label?: LabelExtensionProps;
}

export const SendNotification = () => (
  <Align.Space
    background={3}
    borderShade={4}
    bordered
    rounded={1}
    style={{ padding: "1rem 2rem" }}
  >
    <Text.WithIcon startIcon={<Icon.Channel />} level="p">
      Send Notification
    </Text.WithIcon>
  </Align.Space>
);
