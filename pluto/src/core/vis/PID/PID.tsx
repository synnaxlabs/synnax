// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { Box, CrudeXY, XY } from "@synnaxlabs/x";
import ReactFlow, {
  ReactFlowProvider,
  Viewport as RFViewport,
  useOnViewportChange as useRFOnViewportChange,
  applyEdgeChanges as rfApplyEdgeChanges,
  applyNodeChanges as rfApplyNodeChanges,
  addEdge as rfAddEdge,
  Background as RFBackground,
  SmoothStepEdge as RFSmoothStepEdge,
  Node as RFNode,
  Edge as RFEdge,
  NodeProps as RFNodeProps,
  NodeChange as RFNodeChange,
  EdgeChange as RFEdgeChange,
  Connection as RFConnection,
  ReactFlowProps,
  useReactFlow,
} from "reactflow";

import { Aether } from "@/core/aether/main";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { Button, Pack, Status } from "@/core/std";
import { AetherPID } from "@/core/vis/PID/aether";
import { RenderProp } from "@/util/renderProp";

import "@/core/vis/PID/PID.css";
import "reactflow/dist/style.css";

export interface PIDElementProps {
  elementKey: string;
  position: CrudeXY;
  selected: boolean;
  editable: boolean;
}

export interface PIDEdge {
  key: string;
  source: string;
  target: string;
  sourceHandle?: string | null;
  targetHandle?: string | null;
}

export interface PIDNode {
  key: string;
  position: CrudeXY;
  selected?: boolean;
}

export interface UsePIDProps {
  allowEdit?: boolean;
  initialEdges: PIDEdge[];
  initialNodes: PIDNode[];
}

export const usePID = ({
  initialNodes,
  initialEdges,
  allowEdit = true,
}: UsePIDProps): UsePIDReturn => {
  const [editable, onEditableChange] = useState(allowEdit);
  const [nodes, onNodesChange] = useState<PIDNode[]>(initialNodes);
  const [edges, onEdgesChange] = useState<PIDEdge[]>(initialEdges);

  return {
    edges,
    nodes,
    onNodesChange,
    onEdgesChange,
    editable,
    onEditableChange,
  };
};

export interface UsePIDReturn {
  edges: PIDEdge[];
  nodes: PIDNode[];
  onNodesChange: (cbk: (prev: PIDNode[]) => PIDNode[]) => void;
  onEdgesChange: (cbk: (prev: PIDEdge[]) => PIDEdge[]) => void;
  editable: boolean;
  onEditableChange: (cbk: (prev: boolean) => boolean) => void;
}

export interface PIDProps extends UsePIDReturn {
  children: RenderProp<PIDElementProps>;
}

const translateNodesForward = (
  nodes: PIDNode[],
  editable: boolean
): Array<RFNode<RFNodeData>> =>
  nodes.map((node) => ({
    ...node,
    id: node.key,
    type: "custom",
    data: { editable },
  }));

const translateEdgesForward = (edges: PIDEdge[]): RFEdge[] =>
  edges.map((edge) => ({
    id: edge.key,
    ...edge,
  }));

const translateNodesBackward = (nodes: RFNode[]): PIDNode[] =>
  nodes.map((node) => ({
    key: node.id,
    selected: node.selected,
    ...node,
  }));

const translateEdgesBackward = (edges: RFEdge[]): PIDEdge[] =>
  edges.map((edge) => ({
    key: edge.id,
    ...edge,
  }));

const EDGE_TYPES = { default: RFSmoothStepEdge };

const EDITABLE_PROPS: ReactFlowProps = {
  nodesDraggable: true,
  nodesConnectable: true,
  elementsSelectable: true,
};

const NOT_EDITABLE_PROPS: ReactFlowProps = {
  nodesDraggable: false,
  nodesConnectable: false,
  elementsSelectable: false,
  panOnDrag: false,
  panOnScroll: false,
};

export interface RFNodeData {
  editable: boolean;
}

const PIDCore = Aether.wrap<PIDProps>(
  "PIDCore",
  ({
    aetherKey,
    children,
    onNodesChange,
    onEdgesChange,
    nodes,
    edges,
    onEditableChange,
    editable,
  }): ReactElement => {
    const [{ path }, { error }, setState] = Aether.use({
      aetherKey,
      type: AetherPID.TYPE,
      schema: AetherPID.stateZ,
      initialState: {
        position: XY.ZERO,
        region: Box.ZERO,
      },
    });

    const resizeRef = useResize(
      (box) => setState((prev) => ({ ...prev, region: box })),
      { debounce: 0 }
    );

    const handleViewport = useCallback(
      (viewport: RFViewport): void =>
        setState((prev) => ({ ...prev, position: viewport })),
      []
    );

    useRFOnViewportChange({
      onStart: handleViewport,
      onChange: handleViewport,
      onEnd: handleViewport,
    });

    const Node = useCallback(
      ({ id, xPos, yPos, selected, data: { editable } }: RFNodeProps<RFNodeData>) => {
        return children({
          elementKey: id,
          position: { x: xPos, y: yPos },
          selected,
          editable,
        });
      },
      []
    );

    const nodeTypes = useMemo(() => ({ custom: Node }), []);
    const edges_ = useMemo(() => translateEdgesForward(edges), [edges]);
    const nodes_ = useMemo(
      () => translateNodesForward(nodes, editable),
      [nodes, editable]
    );

    const handleNodesChange = useCallback(
      (changes: RFNodeChange[]) =>
        onNodesChange((prev) =>
          translateNodesBackward(
            rfApplyNodeChanges(changes, translateNodesForward(prev, editable))
          )
        ),
      [onNodesChange, editable]
    );

    const handleEdgesChange = useCallback(
      (changes: RFEdgeChange[]) =>
        onEdgesChange((prev) =>
          translateEdgesBackward(
            rfApplyEdgeChanges(changes, translateEdgesForward(prev))
          )
        ),
      [onEdgesChange]
    );

    const handleConnect = useCallback(
      (conn: RFConnection) =>
        onEdgesChange((prev) =>
          translateEdgesBackward(rfAddEdge(conn, translateEdgesForward(prev)))
        ),
      [onEdgesChange]
    );

    const { fitView } = useReactFlow();

    const editableProps = editable ? EDITABLE_PROPS : NOT_EDITABLE_PROPS;

    if (error != null) {
      return (
        <Aether.Composite path={path}>
          <Status.Text.Centered variant="error" hideIcon level="h4">
            {error}
          </Status.Text.Centered>
        </Aether.Composite>
      );
    }

    return (
      <Aether.Composite path={path}>
        <ReactFlow
          nodes={nodes_}
          edges={edges_}
          nodeTypes={nodeTypes}
          edgeTypes={EDGE_TYPES}
          ref={resizeRef}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          onConnect={handleConnect}
          minZoom={1}
          maxZoom={1}
          panOnScroll={true}
          proOptions={{
            hideAttribution: true,
          }}
          {...editableProps}
        >
          {editable && <RFBackground />}
          <PIDControls
            editable={editable}
            onEditableChange={onEditableChange}
            onFitView={fitView}
          />
        </ReactFlow>
      </Aether.Composite>
    );
  }
);

interface PIDControlsProps {
  editable: boolean;
  onEditableChange: (cbk: (prev: boolean) => boolean) => void;
  onFitView: () => void;
}

const PIDControls = ({
  editable,
  onEditableChange,
  onFitView,
}: PIDControlsProps): ReactElement => {
  return (
    <Pack direction="y" className={CSS.BE("pid", "controls")}>
      <Button.Icon
        onClick={() => onEditableChange((prev) => !prev)}
        variant={editable ? "outlined" : "filled"}
      >
        {editable ? <Icon.EditOff /> : <Icon.Edit />}
      </Button.Icon>
      <Button.Icon onClick={onFitView}>
        <Icon.Expand />
      </Button.Icon>
    </Pack>
  );
};

export const PID = (props: PIDProps): ReactElement => (
  <ReactFlowProvider>
    <PIDCore {...props} />
  </ReactFlowProvider>
);
