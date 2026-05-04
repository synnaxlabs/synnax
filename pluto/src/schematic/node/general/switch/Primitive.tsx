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
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { type Toggle } from "@/schematic/node/common/toggle";

export interface Props extends Omit<Toggle.ButtonProps, "onClick"> {
  onClick?: MouseEventHandler<HTMLElement>;
}

export const Primitive = ({
  enabled = false,
  onClick,
  orientation = "left",
}: Props): ReactElement => (
  <Base.Div orientation={orientation}>
    <BaseInput.Switch value={enabled} onClick={onClick} onChange={() => {}} />
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={0}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={100}
        top={50}
        id="2"
      />
    </Handle.Boundary>
  </Base.Div>
);
