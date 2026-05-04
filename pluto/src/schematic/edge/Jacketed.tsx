import { xy } from "@synnaxlabs/x";

import { Base } from "@/schematic/edge/base";
import { Path } from "@/schematic/edge/path";
import { Segmented } from "@/schematic/edge/segmented";

const JACKET_OFFSET = 6;
const JACKET_OPACITY = 0.7;
const JACKET_STYLE = { opacity: JACKET_OPACITY };

export const jacketedSpec = Segmented.createSpec(
  "jacketed",
  "Jacketed",
  ({ points, color }) => {
    const miters = xy.calculateMiters(points, JACKET_OFFSET);
    const above = points.map((p, i) => xy.translate(p, miters[i]));
    const below = points.map((p, i) => xy.translate(p, xy.scale(miters[i], -1)));
    return (
      <>
        <Base.Base path={Path.rounded(above)} color={color} style={JACKET_STYLE} />
        <Base.Base path={Path.rounded(points)} color={color} />
        <Base.Base path={Path.rounded(below)} color={color} style={JACKET_STYLE} />
      </>
    );
  },
);
