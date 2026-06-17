import { Control } from "@synnaxlabs/pluto";

import { Session } from "@/schematic/session";

export const Legend = () => {
  const legend = Session.useSelectLegend();
  const handleLegendPositionChange = Session.useMoveLegend();
  const handleLegendColorChange = Session.useSetLegendColors();
  if (!legend.visible) return null;
  return (
    <Control.Legend
      position={legend.position}
      onPositionChange={handleLegendPositionChange}
      colors={legend.colors}
      onColorsChange={handleLegendColorChange}
      allowEntryVisibleChange={false}
    />
  );
};
