// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MouseEventHandler, type ReactElement } from "react";

import { Input as BaseInput } from "@/input";
import {
  Div,
  Handle,
  HandleBoundary,
  type ToggleValveButtonProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends Omit<ToggleValveButtonProps, "onClick"> {
  onClick?: MouseEventHandler<HTMLElement>;
}

export const Primitive = ({
  enabled = false,
  onClick,
  orientation = "left",
}: Props): ReactElement => (
  <Div orientation={orientation}>
    <BaseInput.Switch value={enabled} onClick={onClick} onChange={() => {}} />
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={0} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={100} top={50} id="2" />
    </HandleBoundary>
  </Div>
);
