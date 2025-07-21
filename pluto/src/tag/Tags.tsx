// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";

export interface TagsProps extends Align.SpaceProps {
  actions?: ReactNode;
  variant?: "text" | "outlined";
}

export const Tags = ({
  children,
  onClick,
  actions,
  className,
  variant = "outlined",
  ...rest
}: TagsProps) => (
  <Align.Pack {...rest}>
    <Align.Space
      x
      className={CSS(
        onClick && CSS.M("clickable"),
        CSS.M(variant),
        CSS.shade(0),
        CSS.size("medium"),
        className,
      )}
      onClick={onClick}
      align="center"
      size="small"
      style={{ padding: "0rem 0.5rem", minWidth: 300 }}
      rounded
      {...rest}
    >
      {children}
    </Align.Space>
    {actions}
  </Align.Pack>
);
