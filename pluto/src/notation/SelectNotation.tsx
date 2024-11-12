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
import { type Notation } from "@/notation/notation";
import { Button, type ButtonOptionProps, type ButtonProps } from "@/select/Button";

interface Entry {
  key: Notation;
  label: string;
}

const DATA: Entry[] = [
  { key: "standard", label: "Standard" },
  { key: "scientific", label: "Scientific" },
  { key: "engineering", label: "Engineering" },
];

const defaultSelectNotationButton = ({
  key,
  entry,
  onClick,
  selected,
}: ButtonOptionProps<Notation, Entry>): ReactElement => (
  <CoreButton.Button
    key={key}
    variant={selected ? "filled" : "outlined"}
    onClick={onClick}
  >
    {entry.label}
  </CoreButton.Button>
);

export interface SelectNotationProps
  extends Omit<ButtonProps<Notation, Entry>, "data" | "entryRenderKey"> {}

export const SelectNotation = ({
  children = defaultSelectNotationButton,
  ...props
}: SelectNotationProps): ReactElement => (
  <Button {...props} data={DATA}>
    {children}
  </Button>
);
