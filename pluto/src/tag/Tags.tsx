// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tag/Tags.css";

import { Button } from "@/button";
import { CSS } from "@/css";

export interface TagsProps extends Button.ButtonProps<"div"> {}

export const Tags = ({
  children,
  onClick,
  variant = "outlined",
  size = "medium",
  className,
  ...rest
}: TagsProps) => (
  <Button.Button
    className={CSS(CSS.B("tags"), className)}
    el="div"
    variant={variant}
    size={size}
    x
    onClick={onClick}
    align="center"
    gap="small"
    rounded
    {...rest}
  >
    {children}
  </Button.Button>
);
