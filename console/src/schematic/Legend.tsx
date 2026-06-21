import { Control, Schematic } from "@synnaxlabs/pluto";
import { type color, type sticky } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { useSelectLegend } from "@/schematic/selectors";
import { moveLegend, setLegendColors } from "@/schematic/slice";

export const Legend = (): ReactElement | null => {
  const key = Schematic.useKey();
  const { visible, position, colors } = useSelectLegend(key);
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
};
