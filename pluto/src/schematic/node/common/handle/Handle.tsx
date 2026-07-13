// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import {
  Handle as RFHandle,
  type HandleProps as RFHandleProps,
  type Position as RFPosition,
} from "@xyflow/react";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { adjust, smart, swap } from "@/schematic/node/common/handle/position";
import { stopPropagation } from "@/util/event";

export interface HandleProps extends Omit<RFHandleProps, "type" | "position"> {
  orientation: location.Outer;
  location: location.Outer;
  position?: RFPosition;
  preventAutoAdjust?: boolean;
  swap?: boolean;
  left: number;
  top: number;
  id: string;
}

export const Handle = ({
  location,
  orientation,
  preventAutoAdjust,
  left,
  swap: swapPos,
  top,
  style,
  ...rest
}: HandleProps): ReactElement => {
  const adjusted = adjust(top, left, orientation, preventAutoAdjust);
  const mergedStyle = useMemo(
    () => ({ left: `${adjusted.left}%`, top: `${adjusted.top}%`, ...style }),
    [adjusted.left, adjusted.top, style],
  );
  return (
    <RFHandle
      position={swap(smart(location, orientation), !swapPos)}
      {...rest}
      type="source"
      onClick={stopPropagation}
      className={CSS(CSS.B("handle"), CSS.BE("handle", rest.id))}
      style={mergedStyle}
    />
  );
};
