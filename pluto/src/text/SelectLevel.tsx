// Copyright 2023 Synnax Labs, Inc.
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
import { type Level } from "@/text/types";

interface Entry {
  key: Level;
  label: string;
}

export interface SelectLevelProps
  extends Omit<ButtonProps<Level, Entry>, "data" | "entryRenderKey"> {}

const DATA: Entry[] = [
  {
    key: "h1",
    label: "H1",
  },
  {
    key: "h2",
    label: "H2",
  },
  {
    key: "h3",
    label: "H3",
  },
  {
    key: "h4",
    label: "H4",
  },
  {
    key: "h5",
    label: "H5",
  },
  {
    key: "p",
    label: "P",
  },
  {
    key: "small",
    label: "Small",
  },
];

const defaultSelectDirectionButton = ({
  key,
  entry,
  onClick,
  selected,
}: ButtonOptionProps<Level, Entry>): ReactElement => {
  return (
    <CoreButton.Button
      key={key}
      variant={selected ? "filled" : "outlined"}
      onClick={onClick}
    >
      {entry.label}
    </CoreButton.Button>
  );
};

export const SelectLevel = ({
  children = defaultSelectDirectionButton,
  ...props
}: SelectLevelProps): ReactElement => {
  return (
    <Button {...props} data={DATA}>
      {children}
    </Button>
  );
};
