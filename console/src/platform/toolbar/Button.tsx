// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/toolbar/Button.css";

import { Button as Base } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface ButtonProps extends Base.ButtonProps {}

/** An icon action seated in a toolbar header, filling its height. */
export const Button = ({ className, ...rest }: ButtonProps): ReactElement => (
  <Base.Button
    size="medium"
    variant="text"
    className={CSS(CSS.BE("toolbar", "button"), className)}
    {...rest}
  />
);
