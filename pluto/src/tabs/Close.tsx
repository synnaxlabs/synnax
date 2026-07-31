// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon } from "@/icon";

export interface CloseProps extends Button.ButtonProps {}

export const Close = ({
  className,
  children = <Icon.Close />,
  ...rest
}: CloseProps): ReactElement => (
  <Button.Button
    aria-label="Close"
    className={CSS(CSS.BE("tabs", "close"), className)}
    variant="text"
    sharp
    // ARIA tabs have presentational children; the keyboard path is Delete on
    // the focused tab, so the button stays a pointer affordance.
    tabIndex={-1}
    {...rest}
  >
    {children}
  </Button.Button>
);
