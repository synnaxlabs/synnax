// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, Schematic } from "@synnaxlabs/pluto";
import { type color, type sticky } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { useSelectLegend } from "@/schematic/session/selectors";
import { moveLegend, setLegendColors } from "@/schematic/session/slice";

export const Legend = memo((): ReactElement | null => {
  const { visible, position, colors } = useSelectLegend();
  const dispatch = useDispatch();
  const handleLegendPositionChange = useCallback(
    (position: sticky.XY) => dispatch(moveLegend({ key, position })),
    [dispatch, key],
  );
  const handleLegendColorsChange = useCallback(
    (colors: Record<string, color.Color>) => dispatch(setLegendColors({ key, colors })),
    [key, dispatch],
  );
  if (!visible) return null;
  return (
    <Control.Legend
      position={position}
      onPositionChange={handleLegendPositionChange}
      colors={colors}
      onColorsChange={handleLegendColorsChange}
      allowEntryVisibleChange={false}
    />
  );
});
Legend.displayName = "Schematic.Legend";
