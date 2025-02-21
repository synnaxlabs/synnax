// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button, type ButtonProps } from "@/select/Button";
import { type text } from "@/text/core";

interface LevelEntry {
  key: text.Level;
  label: string;
}

export interface SelectLevelProps
  extends Omit<ButtonProps<text.Level, LevelEntry>, "data" | "entryRenderKey"> {}

const DATA: LevelEntry[] = [
  { key: "h2", label: "XL" },
  { key: "h3", label: "L" },
  { key: "h4", label: "M" },
  { key: "h5", label: "S" },
  { key: "small", label: "XS" },
];

export const SelectLevel = (props: SelectLevelProps): ReactElement => (
  <Button {...props} data={DATA} entryRenderKey="label" />
);

interface WeightEntry {
  key: text.Weight;
  label: string;
}

const WEIGHT_DATA: WeightEntry[] = [
  { key: 600, label: "Bold" },
  { key: 500, label: "Medium" },
  { key: 400, label: "Normal" },
  { key: 250, label: "Light" },
];

export interface SelectWeightProps
  extends Omit<ButtonProps<text.Weight, WeightEntry>, "data" | "entryRenderKey"> {}

export const SelectWeight = (props: SelectWeightProps): ReactElement => (
  <Button {...props} data={WEIGHT_DATA} entryRenderKey="label" />
);
