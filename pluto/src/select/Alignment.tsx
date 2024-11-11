// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { type Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { Button, type ButtonOptionProps, type ButtonProps } from "@/select/Button";

interface Entry {
  key: Align.Alignment;
  icon: ReactElement;
}

export interface AlignmentProps
  extends Omit<
    ButtonProps<Align.Alignment, Entry>,
    "data" | "entryRenderKey" | "allowMultiple"
  > {}

const DATA: Entry[] = [
  {
    key: "start",
    icon: <Icon.TextAlign.Left />,
  },
  {
    key: "center",
    icon: <Icon.TextAlign.Center />,
  },
  {
    key: "end",
    icon: <Icon.TextAlign.Right />,
  },
];

const defaultSelectTextAlignmentButton = ({
  key,
  entry,
  onClick,
  selected,
}: ButtonOptionProps<Align.Alignment, Entry>): ReactElement => (
    <CoreButton.Icon
      key={key}
      variant={selected ? "filled" : "outlined"}
      onClick={onClick}
    >
      {entry.icon}
    </CoreButton.Icon>
  );

export const TextAlignment = ({
  children = defaultSelectTextAlignmentButton,
  ...props
}: AlignmentProps): ReactElement => (
    <Button {...props} allowMultiple={false} data={DATA}>
      {children}
    </Button>
  );
