import { Base } from "@/schematic/edge/base";
import { Path } from "@/schematic/edge/path";
import { Segmented } from "@/schematic/edge/segmented";

export const pipeSpec = Segmented.createSpec("pipe", "Pipe", ({ points, color }) => (
  <Base.Base path={Path.rounded(points)} color={color} />
));
