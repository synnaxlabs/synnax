// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, type ButtonProps } from "@/select/Button";
import { type text } from "@/text/core";

export interface WeightEntry {
  key: text.Weight;
  label: string;
}

const WEIGHT_DATA: WeightEntry[] = [
  { key: 600, label: "Bold" },
  { key: 500, label: "Medium" },
  { key: 400, label: "Normal" },
  { key: 250, label: "Light" },
];

export interface WeightProps
  extends Omit<ButtonProps<text.Weight, WeightEntry>, "data" | "entryRenderKey"> {}

export const Weight = (props: WeightProps) => (
  <Button {...props} data={WEIGHT_DATA} entryRenderKey="label" />
);
