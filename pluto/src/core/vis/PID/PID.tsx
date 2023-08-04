// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo, useRef, useState } from "react";

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
import { Edge } from "@/core/vis/PID/Edge";
import { RenderProp } from "@/util/renderProp";

import "@/core/vis/PID/PID.css";
import "reactflow/dist/style.css";

export interface PIDElementProps {
  elementKey: string;
  position: CrudeXY;
  selected: boolean;
  editable: boolean;
}

export interface PIDViewport {
  position: CrudeXY;
  zoom: number;
}

export interface PIDEdge {
  key: string;
  source: string;
  target: string;
  sourceHandle?: string | null;
  targetHandle?: string | null;
  points: CrudeXY[];
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
  initialViewport?: PIDViewport;
}

export const usePID = ({
  initialNodes,
  initialEdges,
  allowEdit = true,
  initialViewport = { position: XY.ZERO, zoom: 1 },
}: UsePIDProps): UsePIDReturn => {
  const [editable, onEditableChange] = useState(allowEdit);
  const [nodes, onNodesChange] = useState<PIDNode[]>(initialNodes);
  const [edges, onEdgesChange] = useState<PIDEdge[]>(initialEdges);
  const [viewport, onViewportChange] = useState<PIDViewport>(initialViewport);

  return {
    viewport,
    onViewportChange,
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
  onNodesChange: (nodes: PIDNode[]) => void;
  onEdgesChange: (edges: PIDEdge[]) => void;
  editable: boolean;
  onEditableChange: (cbk: (prev: boolean) => boolean) => void;
  onViewportChange: (vp: PIDViewport) => void;
  viewport: PIDViewport;
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
  edges.map(({ points, ...edge }) => ({
    id: edge.key,
    data: { points },
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
    points: edge.data?.points ?? [],
    ...edge,
  }));

const translateViewportForward = (viewport: PIDViewport): RFViewport => ({
  ...viewport.position,
  zoom: viewport.zoom,
});

const translateViewportBackward = (viewport: RFViewport): PIDViewport => ({
  position: new XY(viewport).crude,
  zoom: viewport.zoom,
});

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
    viewport,
    onViewportChange,
  }): ReactElement => {
    const [{ path }, { error }, setState] = Aether.use({
      aetherKey,
      type: AetherPID.TYPE,
      schema: AetherPID.stateZ,
      initialState: {
        position: viewport.position,
        region: Box.ZERO,
      },
    });

    const resizeRef = useResize(
      (box) => setState((prev) => ({ ...prev, region: box })),
      { debounce: 0 }
    );

    const handleViewport = useCallback((viewport: RFViewport): void => {
      setState((prev) => ({ ...prev, position: viewport }));
      onViewportChange(translateViewportBackward(viewport));
    }, []);

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
    const edgesRef = useRef(edges);
    const edges_ = useMemo(() => {
      edgesRef.current = edges;
      return translateEdgesForward(edges);
    }, [edges]);
    const nodesRef = useRef(nodes);
    const nodes_ = useMemo(() => {
      nodesRef.current = nodes;
      return translateNodesForward(nodes, editable);
    }, [nodes, editable]);

    const handleNodesChange = useCallback(
      (changes: RFNodeChange[]) =>
        onNodesChange(
          translateNodesBackward(
            rfApplyNodeChanges(
              changes,
              translateNodesForward(nodesRef.current, editable)
            )
          )
        ),
      [onNodesChange, editable]
    );

    const handleEdgesChange = useCallback(
      (changes: RFEdgeChange[]) =>
        onEdgesChange(
          translateEdgesBackward(
            rfApplyEdgeChanges(changes, translateEdgesForward(edgesRef.current))
          )
        ),
      [onEdgesChange]
    );

    const handleConnect = useCallback(
      (conn: RFConnection) =>
        onEdgesChange(
          translateEdgesBackward(
            rfAddEdge(conn, translateEdgesForward(edgesRef.current))
          )
        ),
      [onEdgesChange]
    );

    const handleEdgePointsChange = useCallback(
      (id: string, points: CrudeXY[]) => {
        const next = [...edgesRef.current];
        const index = next.findIndex((e) => e.key === id);
        if (index === -1) return;
        next[index] = { ...next[index], points };
        onEdgesChange(next);
      },
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

    const EDGE_TYPES = useMemo(
      () => ({
        default: (props: any) => {
          return (
            <Edge
              {...props}
              editable={props.data.editable}
              points={props.data.points}
              onPointsChange={(f) => handleEdgePointsChange(props.id, f)}
            />
          );
        },
      }),
      [handleEdgePointsChange]
    );

    return (
      <Aether.Composite path={path}>
        <ReactFlow
          className={CSS(CSS.editable(editable))}
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
          defaultViewport={translateViewportForward(viewport)}
          snapToGrid={true}
          panOnScroll
          selectionOnDrag
          panOnDrag={false}
          selectionKeyCode={null}
          panActivationKeyCode={"Shift"}
          snapGrid={[5, 5]}
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
