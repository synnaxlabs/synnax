// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/fs/PickerRow.css";

import { Button, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface PickerRowProps extends Omit<
  Button.ButtonProps,
  "value" | "onChange" | "onClick"
> {
  /** The displayed path or file name; empty shows the select prompt. */
  value: string;
  /** Opens the picker. */
  onClick: () => void;
}

/** The single-button pick-and-display row InputPath and InputFile share. */
export const PickerRow = ({
  value,
  onClick,
  ...rest
}: PickerRowProps): ReactElement => (
  <Button.Button
    level="small"
    className={CSS.B("picker-row")}
    variant="outlined"
    justify="start"
    onClick={onClick}
    size="medium"
    textColor={9}
    weight={450}
    {...rest}
  >
    {primitive.isNonZero(value) ? (
      <>
        <Icon.Attachment className={CSS.BE("picker-row", "icon")} />
        {value}
      </>
    ) : (
      "Select file"
    )}
  </Button.Button>
);
