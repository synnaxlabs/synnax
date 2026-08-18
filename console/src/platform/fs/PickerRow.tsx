// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/fs/PickerRow.css";

import { Button, Flex, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface PickerRowProps extends Omit<
  Flex.BoxProps,
  "value" | "onChange" | "onClick"
> {
  /** The displayed path or file name; empty shows the placeholder. */
  value: string;
  /** Opens the picker. */
  onClick: () => void;
}

/** The packed display-and-pick row InputPath and InputFile share. */
export const PickerRow = ({
  value,
  onClick,
  ...rest
}: PickerRowProps): ReactElement => (
  <Flex.Box pack className={CSS.B("picker-row")} borderColor={6} {...rest}>
    <Button.Button
      level="small"
      className={CSS.B("path")}
      variant="outlined"
      grow
      onClick={onClick}
      size="medium"
      textColor={9}
      weight={450}
    >
      {primitive.isNonZero(value) ? (
        <>
          <Icon.Attachment className={CSS.BE("picker-row", "icon")} />
          {value}
        </>
      ) : (
        "No file selected"
      )}
    </Button.Button>
    <Button.Button variant="outlined" className={CSS.B("select")} onClick={onClick}>
      Select file
    </Button.Button>
  </Flex.Box>
);
