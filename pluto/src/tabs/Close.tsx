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

/** Props for {@link Close}. */
export interface CloseProps extends Button.CloseProps {}

/** A close button sized and placed for use inside a {@link Tab}. */
export const Close = ({ className, ...rest }: CloseProps): ReactElement => (
  <Button.Close className={CSS(CSS.BE("tabs", "close"), className)} {...rest} />
);
