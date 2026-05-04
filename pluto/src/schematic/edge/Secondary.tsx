import { Base } from "@/schematic/edge/base";
import { Path } from "@/schematic/edge/path";
import { Segmented } from "@/schematic/edge/segmented";

export const secondarySpec = Segmented.createSpec(
  "secondary",
  "Secondary",
  ({ points, color }) => (
    <Base.Base
      path={Path.rounded(points)}
      color={color}
      style={{ strokeDasharray: "12,4,4" }}
    />
  ),
);
