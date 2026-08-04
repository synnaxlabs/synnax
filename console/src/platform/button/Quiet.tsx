// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/button/Quiet.css";

import { Button } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface QuietProps extends Button.ButtonProps {}

/** A text button that rests muted and brightens on hover. */
export const Quiet = ({ className, ...rest }: QuietProps): ReactElement => (
  <Button.Button
    variant="text"
    className={CSS(CSS.B("quiet-btn"), className)}
    {...rest}
  />
);
