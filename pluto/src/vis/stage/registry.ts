// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";
import { z } from "zod";

import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Theming } from "@/theming";
import { ButtonForm } from "@/vis/schematic/Forms";
import { type SymbolFormProps } from "@/vis/stage/Forms";
import {
  Button,
  ButtonPreview,
  type ButtonProps,
  CaseStatement,
  type IfStatementProps,
  type LabelExtensionProps,
  SendNotification,
  type SendNotificationProps,
  Source,
  type SourceProps,
  type SymbolProps,
} from "@/vis/stage/Symbols";

export interface Spec<P extends object> {
  key: Variant;
  name: string;
  Form: FC<SymbolFormProps>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<SymbolProps<P>>;
  zIndex: number;
}

const Z_INDEX_UPPER = 4;

const VARIANTS = ["button", "source", "ifStatement", "sendNotification"] as const;

export const variantZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof variantZ>;

const ZERO_PROPS = { orientation: "left" as const, scale: 1 };

const ZERO_BOOLEAN_SINK_PROPS = {
  ...ZERO_PROPS,
  control: { show: true },
  sink: telem.sinkPipeline("boolean", {
    connections: [{ from: "setpoint", to: "setter" }],
    segments: {
      setter: control.setChannelValue({ channel: 0 }),
      setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
    },
    inlet: "setpoint",
  }),
};

type zeroLabelReturn = { label: LabelExtensionProps };

const zeroLabel = (label: string): zeroLabelReturn => ({
  label: {
    label,
    level: "p",
    orientation: "top",
    maxInlineSize: 150,
    align: "center",
    direction: "x",
  },
});

const button: Spec<ButtonProps> = {
  name: "Button",
  key: "button",
  Symbol: Button,
  Form: ButtonForm,
  Preview: ButtonPreview,
  defaultProps: (t) => ({
    color: t.colors.primary.z,
    ...zeroLabel("Button"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    onClickDelay: 0,
    scale: null,
  }),
  zIndex: Z_INDEX_UPPER,
};

const source: Spec<SourceProps> = {
  name: "Source",
  key: "source",
  Symbol: Source,
  Form: ButtonForm,
  Preview: ButtonPreview,
  defaultProps: (t) => ({
    color: t.colors.primary.z,
    ...zeroLabel("Button"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    onClickDelay: 0,
    scale: null,
  }),
  zIndex: Z_INDEX_UPPER,
};

const ifStatement: Spec<IfStatementProps> = {
  name: "If",
  key: "ifStatement",
  Symbol: CaseStatement,
  Form: ButtonForm,
  Preview: ButtonPreview,
  defaultProps: (t) => ({
    color: t.colors.primary.z,
    ...zeroLabel("If"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    onClickDelay: 0,
    scale: null,
  }),
  zIndex: Z_INDEX_UPPER,
};

const sendNotification: Spec<SendNotificationProps> = {
  name: "Send Notification",
  key: "sendNotification",
  Symbol: SendNotification,
  Form: ButtonForm,
  Preview: ButtonPreview,
  defaultProps: (t) => ({
    color: t.colors.primary.z,
    ...zeroLabel("Send Notification"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    onClickDelay: 0,
    scale: null,
  }),
  zIndex: Z_INDEX_UPPER,
};

export const SYMBOLS: Record<Variant, Spec<any>> = {
  button,
  source,
  ifStatement,
  sendNotification,
};
