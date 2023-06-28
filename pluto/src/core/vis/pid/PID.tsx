// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FC, ReactElement, useCallback, useMemo, useState } from "react";

import { Box, XY } from "@synnaxlabs/x";
import ReactFlow, {
  Handle,
  NodeProps,
  Position,
  ReactFlowProvider,
  Viewport,
  useOnViewportChange,
  applyEdgeChanges,
  applyNodeChanges,
  addEdge,
  Background,
  SmoothStepEdge,
  Controls,
  Edge,
} from "reactflow";

import { Aether } from "@/core/aether/main";
import { useResize } from "@/core/hooks";
import { AetherPID } from "@/core/vis/pid/aether";

/** The props for an element in the PID. */
export type PIDElementProps<P extends unknown = unknown> = P & {
  id: string;
  position: XY;
  selected: boolean;
  editable: boolean;
};

/** An element in the PID that accepts PIDElementProps */
export type PIDElement<P extends unknown = unknown> = FC<PIDElementProps<P>>;

/** A registry containng all PID element types. */
export type PIDElementRegistry = Record<string, PIDElement>;

export interface UsePIDReturn {
  elements: PIDElementProps[];
  reigstry: PIDElementRegistry;
  onConnect: (params: Edge) => void;
}

const PIDInternal = (): ReactElement => {
  const nodeType = useMemo(() => ({ value: ValueNode, valve: ValveNode }), []);
  const edgeType = useMemo(() => ({ default: SmoothStepEdge }), []);

  const [{ path }, , setState] = Aether.useStateful({
    type: AetherPID.TYPE,
    schema: AetherPID.stateZ,
    initialState: {
      position: XY.ZERO,
      region: Box.ZERO,
    },
  });

  const [nodes, setNodes] = useState(n);
  const [edges, setEdges] = useState([]);

  const onNodesChange = useCallback(
    (changes) => setNodes((nds) => applyNodeChanges(changes, nds)),
    []
  );
  const onEdgesChange = useCallback(
    (changes) => setEdges((eds) => applyEdgeChanges(changes, eds)),
    []
  );

  const onConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    []
  );

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
    <Aether.Composite path={path}>
      <ReactFlow
        nodeTypes={nodeType}
        nodes={nodes}
        edges={edges}
        onConnect={onConnect}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        ref={resizeRef}
        edgeTypes={edgeType}
        minZoom={1}
        maxZoom={1}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </Aether.Composite>
  );
};

export const PID = (): ReactElement => {
  return (
    <ReactFlowProvider>
      <PIDInternal />
    </ReactFlowProvider>
  );
};
