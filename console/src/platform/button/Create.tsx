// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/button/Create.css";

import { Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Quiet, type QuietProps } from "@/platform/button/Quiet";
import { CSS } from "@/platform/css";

export interface CreateProps extends Omit<QuietProps, "size"> {
  /** Row scale matching the list the button sits under. */
  size?: "small" | "medium" | "large";
}

/** A pinned create row styled to read as a list's last row. */
export const Create = ({
  className,
  size = "medium",
  children,
  ...rest
}: CreateProps): ReactElement => (
  <Quiet
    full="x"
    justify="start"
    className={CSS(CSS.B("create-btn"), CSS.M(size), className)}
    {...rest}
  >
    <Icon.Add />
    {children}
  </Quiet>
);
