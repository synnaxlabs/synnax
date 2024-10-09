// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button as CoreButton } from "@/button";
import { Button, type ButtonOptionProps, type ButtonProps } from "@/select/Button";

interface Entry {
  key: direction.Direction;
  icon: ReactElement;
}

export interface DirectionProps
  extends Omit<
    ButtonProps<direction.Direction, Entry>,
    "data" | "entryRenderKey" | "allowMultiple"
  > {}

const DATA: Entry[] = [
  {
    key: "x",
    icon: <Icon.Arrow.Right />,
  },
  {
    key: "y",
    icon: <Icon.Arrow.Up />,
  },
];

const defaultSelectDirectionButton = ({
  key,
  entry,
  onClick,
  selected,
}: ButtonOptionProps<direction.Crude, Entry>): ReactElement => (
  <CoreButton.Icon
    key={key}
    variant={selected ? "filled" : "outlined"}
    onClick={onClick}
  >
    {entry.icon}
  </CoreButton.Icon>
);

export const Direction = ({
  children = defaultSelectDirectionButton,
  ...props
}: DirectionProps): ReactElement => (
  <Button {...props} allowMultiple={false} data={DATA}>
    {children}
  </Button>
);
