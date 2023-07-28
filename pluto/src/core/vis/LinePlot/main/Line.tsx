// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useEffect } from "react";

import { Line as CoreLine, LineProps as CoreLineProps } from "../../Line";

import { useLinePlotContext } from "./LinePlot";

import { Aether } from "@/core/aether/main";

export interface LineProps extends CoreLineProps {}

export const Line = Aether.wrap<LineProps>(
  "Line",
  ({ aetherKey, color, label = "", ...props }): ReactElement => {
    const { setLine, removeLine } = useLinePlotContext("Line");
    useEffect(() => {
      setLine({
        key: aetherKey,
        color,
        label,
      });
      return () => removeLine(aetherKey);
    }, [label, color]);
    return <CoreLine aetherKey={aetherKey} color={color} label={label} {...props} />;
  }
);
