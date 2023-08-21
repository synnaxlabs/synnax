// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { CrudeDirection, Direction } from "@synnaxlabs/x";

import { Button } from "@/button";
import { Button, ButtonOptionProps, ButtonProps } from "@/select/Button";

interface Entry {
  key: CrudeDirection;
  icon: ReactElement;
}

export interface DirectionProps
  extends Omit<ButtonProps<CrudeDirection, Entry>, "data" | "entryRenderKey"> {}

const DATA = [
  {
    key: Direction.X.crude,
    icon: <Icon.Arrow.Right />,
  },
  {
    key: Direction.Y.crude,
    icon: <Icon.Arrow.Down />,
  },
];

const defaultSelectDirectionButton = ({
  key,
  entry,
  onClick,
  selected,
}: ButtonOptionProps<CrudeDirection, Entry>): ReactElement => {
  return (
    <Button.Icon key={key} variant={selected ? "filled" : "outlined"} onClick={onClick}>
      {entry.icon}
    </Button.Icon>
  );
};

export const SelectDirection = ({
  children = defaultSelectDirectionButton,
  ...props
}: DirectionProps): ReactElement => {
  return (
    <Button {...props} data={DATA}>
      {children}
    </Button>
  );
};
