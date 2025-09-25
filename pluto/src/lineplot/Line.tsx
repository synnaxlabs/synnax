// Copyright 2025 Synnax Labs, Inc.
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
import { useContext } from "@/lineplot/LinePlot";
import { Line as Core } from "@/vis/line";

export interface LineProps extends Core.LineProps, Aether.ComponentProps {
  legendGroup: string;
}

export const Line = ({
  aetherKey,
  color,
  label = "",
  legendGroup,
  visible = true,
  ...rest
}: LineProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { setLine, removeLine } = useContext("Line");
  useEffect(() => {
    setLine({ key: cKey, color, label, visible, legendGroup });
    return () => removeLine(cKey);
  }, [label, color, visible, legendGroup]);
  return (
    <Core.Line
      aetherKey={cKey}
      color={color}
      label={label}
      visible={visible}
      {...rest}
    />
  );
};
