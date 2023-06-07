// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo } from "react";

import { Box, XY } from "@synnaxlabs/x";
import ReactFlow, {
  NodeProps,
  ReactFlowProvider,
  Viewport,
  useOnViewportChange,
} from "reactflow";

import { Virtual } from "@/core/virtual/main";
import { useResize } from "@/core/hooks";
import { Value } from "@/core/vis/pid/Value/main";
import { PID as WorkerPID, PIDState as WorkerPIDState } from "@/core/vis/pid/worker";
import { usePointTelem } from "@/telem/useStaticTelem";

const ValueNode = (props: NodeProps): ReactElement => {
  const telem = usePointTelem(12000);
  return (
    <Value label="Regen PT" telem={telem} position={new XY(props.xPos, props.yPos)} />
  );
};

const nodes = [{ id: "node-1", type: "value", position: { x: 250, y: 5 }, data: {} }];

const PIDInternal = (): ReactElement => {
  const nodeType = useMemo(() => ({ value: ValueNode }), []);
  const {
    path,
    state: [, setState],
  } = Virtual.useComponent<WorkerPIDState>(WorkerPID.TYPE, {
    position: XY.ZERO,
    region: Box.ZERO,
  });
  const resizeRef = useResize((box) => {
    setState((prev) => ({ ...prev, region: box }));
  }, {});

  const handleViewport = useCallback((viewport: Viewport): void => {
    setState((prev) => ({ ...prev, position: new XY(viewport.x, viewport.y) }));
  }, []);

  useOnViewportChange({
    onStart: handleViewport,
    onChange: handleViewport,
    onEnd: handleViewport,
  });

  return (
    <Virtual.Composite path={path}>
      <ReactFlow
        nodeTypes={nodeType}
        nodes={nodes}
        ref={resizeRef}
        minZoom={1}
        maxZoom={1}
      />
      ;
    </Virtual.Composite>
  );
};

export const PID = (): ReactElement => {
  return (
    <ReactFlowProvider>
      <PIDInternal />
    </ReactFlowProvider>
  );
};
