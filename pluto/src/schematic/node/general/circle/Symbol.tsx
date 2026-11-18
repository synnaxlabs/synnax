// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Circle } from "@/schematic/node/general/circle/Primitive";

export interface SymbolProps extends Omit<schematic.CircleNodeConfig, "label"> {
  className?: string;
}

/** Renders the circle with its colors resolved for the theme. */
export const Symbol = ({
  strokeColor,
  fillColor,
  ...rest
}: SymbolProps): ReactElement => {
  const resolve = Color.useResolve();
  return (
    <Circle
      {...rest}
      strokeColor={resolve(strokeColor)}
      fillColor={resolve(fillColor)}
    />
  );
};
