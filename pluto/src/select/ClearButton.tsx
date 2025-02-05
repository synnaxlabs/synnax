// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";

export const ClearButton = (
  props: Omit<Button.IconProps, "children">,
): ReactElement => (
  <Button.Icon className={CSS.BE("select", "clear")} variant="outlined" {...props}>
    <Icon.Close aria-label="clear" />
  </Button.Icon>
);
