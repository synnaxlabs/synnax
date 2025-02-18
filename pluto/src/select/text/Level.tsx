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

export interface LevelEntry {
  key: text.Level;
  label: string;
}

export interface LevelProps
  extends Omit<ButtonProps<text.Level, LevelEntry>, "data" | "entryRenderKey"> {}

const DATA: LevelEntry[] = [
  { key: "h2", label: "XL" },
  { key: "h3", label: "L" },
  { key: "h4", label: "M" },
  { key: "h5", label: "S" },
  { key: "small", label: "XS" },
];

export const Level = (props: LevelProps) => (
  <Button {...props} data={DATA} entryRenderKey="label" />
);
