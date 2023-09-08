// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { type CrudeDirection, Direction as XDirection } from "@synnaxlabs/x";

import { Button as CoreButton } from "@/button";
import { Button, type ButtonOptionProps, type ButtonProps } from "@/select/Button";

interface Entry {
  key: CrudeDirection;
  icon: ReactElement;
}

export interface DirectionProps
  extends Omit<ButtonProps<CrudeDirection, Entry>, "data" | "entryRenderKey"> {}

const DATA = [
  {
    key: XDirection.X.crude,
    icon: <Icon.Arrow.Right />,
  },
  {
    key: XDirection.Y.crude,
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
    <CoreButton.Icon
      key={key}
      variant={selected ? "filled" : "outlined"}
      onClick={onClick}
    >
      {entry.icon}
    </CoreButton.Icon>
  );
};

export const Direction = ({
  children = defaultSelectDirectionButton,
  ...props
}: DirectionProps): ReactElement => {
  return (
    <Button {...props} data={DATA}>
      {children}
    </Button>
  );
};
