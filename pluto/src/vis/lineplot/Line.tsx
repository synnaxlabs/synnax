// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect } from "react";

import { type Aether } from "@/aether";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Line as Core } from "@/vis/line";
import { useContext } from "@/vis/lineplot/LinePlot";

export interface LineProps extends Core.LineProps, Aether.CProps {}

export const Line = ({
  aetherKey,
  color,
  label = "",
  ...props
}: LineProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { setLine, removeLine } = useContext("Line");
  useEffect(() => {
    setLine({ key: cKey, color, label });
    return () => removeLine(cKey);
  }, [label, color]);
  return <Core.Line aetherKey={cKey} color={color} label={label} {...props} />;
};
