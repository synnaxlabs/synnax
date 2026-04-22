// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as colorUtil } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useMemo } from "react";

import { type Aether } from "@/aether";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { type SubGroup, useContext } from "@/lineplot/LinePlot";
import { Line as Base } from "@/vis/line";

const DIMMED_OPACITY = 0.15;

export interface LineProps extends Base.LineProps, Aether.ComponentProps {
  legendGroup: string;
  subGroup?: SubGroup;
}

export const Line = ({
  aetherKey,
  color,
  label = "",
  legendGroup,
  subGroup,
  visible = true,
  ...rest
}: LineProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { setLine, removeLine, highlightedSubGroup } = useContext("Line");
  useEffect(() => {
    setLine({ key: cKey, color, label, visible, legendGroup, subGroup });
    return () => removeLine(cKey);
  }, [label, color, visible, legendGroup, subGroup]);

  const effectiveColor = useMemo(() => {
    if (highlightedSubGroup == null || subGroup == null) return color;
    if (subGroup.key === highlightedSubGroup) return color;
    return colorUtil.setAlpha(color, DIMMED_OPACITY);
  }, [color, highlightedSubGroup, subGroup]);

  return (
    <Base.Line
      aetherKey={cKey}
      color={effectiveColor}
      label={label}
      visible={visible}
      {...rest}
    />
  );
};
