import { box, color, location } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import { type ReactElement } from "react";

import { Base } from "@/schematic/edge/base";
import { Path } from "@/schematic/edge/path";
import { Segmented } from "@/schematic/edge/segmented";
import { type Diagram } from "@/vis/diagram";

export const ConnectionLine = ({
  source,
  target,
  style,
  status,
}: Diagram.ConnectionLineProps): ReactElement => {
  const connectedHandle = document.querySelector(".react-flow__handle-connecting");
  const toNodeHandle = connectedHandle?.className.match(/react-flow__handle-(\w+)/);
  if (toNodeHandle != null) {
    const res = location.outerZ.safeParse(toNodeHandle[1]);
    if (res.success) target.orientation = res.data;
  }
  const flow = useReactFlow();
  const conn = Segmented.createConnector({
    sourcePos: source.position,
    targetPos: target.position,
    sourceOrientation: source.orientation,
    targetOrientation: target.orientation,
    sourceBox: box.ZERO,
    targetBox: box.ZERO,
  });
  const points = Segmented.segmentsToPoints(
    source.position,
    conn,
    flow.getZoom(),
    false,
  );
  return (
    <Base.Base
      path={Path.rounded(points)}
      color={color.ZERO}
      style={{
        ...style,
        stroke: color.cssString(
          status === "invalid" ? "var(--pluto-error-z)" : "var(--pluto-gray-l11)",
        ),
        strokeWidth: 2,
        fill: "none",
      }}
    />
  );
};
