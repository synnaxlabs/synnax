// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Modal.css";

import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";

export interface BackgroundProps extends Align.SpaceProps {
  visible: boolean;
}

export const Background = ({
  children,
  visible,
  ...rest
}: BackgroundProps): ReactElement => (
  <Align.Space
    className={CSS(CSS.BE("modal", "bg"), CSS.visible(visible))}
    empty
    align="center"
    {...rest}
  >
    {children}
  </Align.Space>
);
