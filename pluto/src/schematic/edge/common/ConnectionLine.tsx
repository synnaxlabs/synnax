// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import { type CSSProperties, type ReactElement } from "react";

import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { type Diagram } from "@/vis/diagram";

const STYLE: CSSProperties = { strokeWidth: 2 };

export const ConnectionLine = ({
  source,
  target,
  sourceBox,
  targetBox,
  status,
}: Diagram.ConnectionLineProps): ReactElement => {
  const connectedHandle = document.querySelector(".react-flow__handle-connecting");
  const toNodeHandle = connectedHandle?.className.match(/react-flow__handle-(\w+)/);
  if (toNodeHandle != null) {
    const res = location.outerZ.safeParse(toNodeHandle[1]);
    if (res.success) target.orientation = res.data;
  }
  const flow = useReactFlow();
  const conn = Segmented.createConnector({ source, target, sourceBox, targetBox });
  const points = Segmented.segmentsToPoints(
    source.position,
    conn,
    flow.getZoom(),
    false,
  );
  const colorVar =
    status == "invalid" ? "var(--pluto-error-z)" : "var(--pluto-gray-l11)";
  return <Base.Base path={Path.rounded(points)} color={colorVar} style={STYLE} />;
};
