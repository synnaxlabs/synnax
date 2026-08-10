// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/vis/Controls.css";

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface ControlsProps extends Flex.BoxProps {}

/** Marks a control as exempt from the hover fade, keeping it visible while an
 * active mode (acquired control, paused plotting) must stay indicated. May sit
 * at any depth inside Controls; its siblings still fade. */
export const CONTROLS_PINNED_CLASS = CSS.BM("controls", "pinned");

export const Controls = ({ className, ...rest }: ControlsProps): ReactElement => (
  <Flex.Box className={CSS(CSS.B("controls"), className)} gap="small" {...rest} />
);
