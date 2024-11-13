// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button as CoreButton } from "@/button";
import { Button, type ButtonOptionProps, type ButtonProps } from "@/select/Button";
import { type text } from "@/text/core";

interface Entry {
  key: text.Level;
  label: string;
}

export interface SelectLevelProps
  extends Omit<ButtonProps<text.Level, Entry>, "data" | "entryRenderKey"> {}

const DATA: Entry[] = [
  { key: "h2", label: "XL" },
  { key: "h3", label: "L" },
  { key: "h4", label: "M" },
  { key: "h5", label: "S" },
  { key: "small", label: "XS" },
];

const defaultSelectDirectionButton = ({
  key,
  entry,
  onClick,
  selected,
}: ButtonOptionProps<text.Level, Entry>): ReactElement => (
  <CoreButton.Button
    key={key}
    variant={selected ? "filled" : "outlined"}
    onClick={onClick}
  >
    {entry.label}
  </CoreButton.Button>
);

export const SelectLevel = ({
  children = defaultSelectDirectionButton,
  ...props
}: SelectLevelProps): ReactElement => (
  <Button {...props} data={DATA}>
    {children}
  </Button>
);
