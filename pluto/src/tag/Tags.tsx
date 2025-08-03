// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@/button";

export interface TagsProps extends Button.ButtonProps<"div"> {
  variant?: "text" | "outlined";
}

export const Tags = ({
  children,
  onClick,
  variant = "outlined",
  size = "medium",
  ...rest
}: TagsProps) => (
  <Button.Button
    el="div"
    variant={variant}
    size={size}
    x
    onClick={onClick}
    align="center"
    gap="small"
    style={{ padding: "0rem 0.5rem", paddingRight: "2rem" }}
    rounded
    {...rest}
  >
    {children}
  </Button.Button>
);
