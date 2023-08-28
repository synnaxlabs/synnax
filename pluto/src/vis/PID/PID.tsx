// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ReactElement,
  createContext,
  useCallback,
  useMemo,
  useRef,
  useState,
  useContext as reactUseContext,
  ComponentPropsWithoutRef,
  useEffect,
  memo,
  forwardRef,
} from "react";

import { Icon } from "@synnaxlabs/media";
import { Box, CrudeXY, Deep, XY, XYLocation } from "@synnaxlabs/x";
import ReactFlow, {
  ReactFlowProvider,
  Viewport as RFViewport,
  useOnViewportChange as useRFOnViewportChange,
  applyEdgeChanges as rfApplyEdgeChanges,
  applyNodeChanges as rfApplyNodeChanges,
  addEdge as rfAddEdge,
  Background as RFBackground,
  Edge as RFEdge,
  NodeProps as RFNodeProps,
  NodeChange as RFNodeChange,
  EdgeChange as RFEdgeChange,
  Connection as RFConnection,
  ReactFlowProps,
  useReactFlow,
  useViewport,
  ConnectionMode,
  updateEdge,
} from "reactflow";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useResize } from "@/hooks";
import { Status } from "@/status";
import { Text } from "@/text";
import { RenderProp } from "@/util/renderProp";
import { pid } from "@/vis/pid/aether";
import { Edge as PlutoEdge } from "@/vis/pid/edge";

import {
  Edge,
  Node,
  Viewport,
  edgeConverter,
  nodeConverter,
  translateEdgesForward,
  translateNodesForward,
  translateViewportBackward,
  translateViewportForward,
} from "./types";

import "@/vis/pid/PID.css";
import "reactflow/dist/style.css";

export interface ElementProps {
  elementKey: string;
  position: CrudeXY;
  zoom: number;
  selected: boolean;
  editable: boolean;
}

export interface UseProps {
  allowEdit?: boolean;
  initialEdges: Edge[];
  initialNodes: Node[];
  initialViewport?: Viewport;
}

export const use = ({
  initialNodes,
  initialEdges,
  allowEdit = true,
  initialViewport = { position: XY.ZERO, zoom: 1 },
}: UseProps): UseReturn => {
  const [editable, onEditableChange] = useState(allowEdit);
  const [nodes, onNodesChange] = useState<Node[]>(initialNodes);
  const [edges, onEdgesChange] = useState<Edge[]>(initialEdges);
  const [viewport, onViewportChange] = useState<Viewport>(initialViewport);

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

export interface UseReturn {
  edges: Edge[];
  nodes: Node[];
  onNodesChange: (nodes: Node[]) => void;
  onEdgesChange: (edges: Edge[]) => void;
  editable: boolean;
  onEditableChange: (v: boolean) => void;
  onViewportChange: (vp: Viewport) => void;
  viewport: Viewport;
}

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

export interface PIDProps
  extends UseReturn,
    Omit<ComponentPropsWithoutRef<"div">, "onError"> {}

interface ContextValue {
  editable: boolean;
  onEditableChange: (v: boolean) => void;
  registerNodeRenderer: (renderer: RenderProp<ElementProps>) => void;
}

const Context = createContext<ContextValue>({
  editable: true,
  onEditableChange: () => {},
  registerNodeRenderer: (renderer: RenderProp<ElementProps>) => {},
});

export const useContext = (): ContextValue => reactUseContext(Context);

export interface NodeRendererProps {
  children: RenderProp<ElementProps>;
}

export const NodeRenderer = memo(
  ({ children }: NodeRendererProps): ReactElement | null => {
    const { registerNodeRenderer } = useContext();
    useEffect(() => registerNodeRenderer(children), [registerNodeRenderer, children]);
    return null;
  }
);
NodeRenderer.displayName = "NodeRenderer";

const Core = Aether.wrap<PIDProps>(
  pid.PID.TYPE,
  // eslint-disable-next-line react/display-name
  forwardRef(
    (
      {
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
        ...props
      },
      ref
    ): ReactElement => {
      const [{ path }, { error }, setState] = Aether.use({
        aetherKey,
        type: pid.PID.TYPE,
        schema: pid.PID.stateZ,
        initialState: {
          position: viewport.position,
          region: Box.ZERO,
          zoom: viewport.zoom,
        },
      });

      const resizeRef = useResize(
        (box) => {
          setState((prev) => ({ ...prev, region: box }));
        },
        { debounce: 0 }
      );

      // For some reason, react flow repeatedly calls onViewportChange with the same
      // paramters, so we do a need equality check to prevent unnecessary re-renders.
      const vpRef = useRef<RFViewport | null>(null);
      const handleViewport = useCallback(
        (viewport: RFViewport): void => {
          if (vpRef.current != null && Deep.equal(viewport, vpRef.current)) return;
          vpRef.current = viewport;
          setState((prev) => ({ ...prev, position: viewport, zoom: viewport.zoom }));
          onViewportChange(translateViewportBackward(viewport));
        },
        [setState, onViewportChange]
      );

      useRFOnViewportChange({
        onStart: handleViewport,
        onChange: handleViewport,
        onEnd: handleViewport,
      });

      const [renderer, setRenderer] = useState<RenderProp<ElementProps>>(
        () => () => null
      );

      const registerNodeRenderer = useCallback(
        (renderer: RenderProp<ElementProps>) => setRenderer(() => renderer),
        []
      );

      const Node = useCallback(
        ({ id, xPos, yPos, selected }: RFNodeProps) => {
          const { zoom } = useViewport();
          const { editable } = useContext();
          return renderer({
            elementKey: id,
            position: { x: xPos, y: yPos },
            zoom,
            selected,
            editable,
          });
        },
        [renderer]
      );

      const nodeTypes = useMemo(() => ({ custom: Node }), [Node]);
      const edgesRef = useRef(edges);
      const edges_ = useMemo(() => {
        edgesRef.current = edges;
        return translateEdgesForward(edges);
      }, [edges]);
      const nodesRef = useRef(nodes);
      const nodes_ = useMemo(() => {
        nodesRef.current = nodes;
        return translateNodesForward(nodes);
      }, [nodes]);

      const handleNodesChange = useCallback(
        (changes: RFNodeChange[]) =>
          onNodesChange(
            nodeConverter(nodesRef.current, (n) => rfApplyNodeChanges(changes, n))
          ),
        [onNodesChange]
      );

      const handleEdgesChange = useCallback(
        (changes: RFEdgeChange[]) =>
          onEdgesChange(
            edgeConverter(edgesRef.current, (e) => rfApplyEdgeChanges(changes, e))
          ),
        [onEdgesChange]
      );

      const handleEdgeUpdate = useCallback(
        (oldEdge: RFEdge, newConnection: RFConnection) =>
          onEdgesChange(
            edgeConverter(edgesRef.current, (e) =>
              updateEdge(oldEdge, newConnection, e)
            )
          ),
        []
      );

      const handleConnect = useCallback(
        (conn: RFConnection) =>
          onEdgesChange(edgeConverter(edgesRef.current, (e) => rfAddEdge(conn, e))),
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

      const editableProps = editable ? EDITABLE_PROPS : NOT_EDITABLE_PROPS;

      const EDGE_TYPES = useMemo(
        () => ({
          default: (props: any) => (
            <PlutoEdge
              {...props}
              editable={props.data.editable}
              points={props.data.points}
              color={props.data.color}
              onPointsChange={(f) => handleEdgePointsChange(props.id, f)}
            />
          ),
        }),
        [handleEdgePointsChange]
      );

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
        <Context.Provider value={{ editable, onEditableChange, registerNodeRenderer }}>
          <Aether.Composite path={path}>
            <ReactFlow
              className={CSS(CSS.B("pid"), CSS.editable(editable))}
              nodes={nodes_}
              edges={edges_}
              nodeTypes={nodeTypes}
              edgeTypes={EDGE_TYPES}
              ref={resizeRef}
              onNodesChange={handleNodesChange}
              onEdgesChange={handleEdgesChange}
              onConnect={handleConnect}
              onEdgeUpdate={handleEdgeUpdate}
              defaultViewport={translateViewportForward(viewport)}
              snapToGrid={true}
              panOnScroll
              selectionOnDrag
              panOnDrag={false}
              minZoom={0.5}
              maxZoom={1}
              selectionKeyCode={null}
              panActivationKeyCode={"Shift"}
              connectionMode={ConnectionMode.Loose}
              snapGrid={[5, 5]}
              proOptions={{
                hideAttribution: true,
              }}
              {...editableProps}
              {...props}
            >
              {children}
            </ReactFlow>
          </Aether.Composite>
        </Context.Provider>
      );
    }
  )
);

export const Background = (): ReactElement | null => {
  const { editable } = useContext();
  return editable ? <RFBackground /> : null;
};

export interface ControlsProps extends Align.PackProps {}

export const Controls = ({ children, ...props }: ControlsProps): ReactElement => (
  <Align.Pack direction="y" className={CSS.BE("pid", "controls")} {...props}>
    {children}
  </Align.Pack>
);

export type ToggleEditControlProps = Omit<Button.ToggleIconProps, "value" | "onChange">;

export const ToggleEditControl = ({
  onClick,
  ...props
}: ToggleEditControlProps): ReactElement => {
  const { editable, onEditableChange } = useContext();
  return (
    <Button.ToggleIcon
      onChange={() => onEditableChange(!editable)}
      value={editable}
      tooltipLocation={XYLocation.RIGHT_CENTER.crude}
      tooltip={
        editable ? (
          <Text.Text level="small">Enable edit mode</Text.Text>
        ) : (
          <Text.Text level="small">Disable edit mode</Text.Text>
        )
      }
      {...props}
    >
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
    </Button.ToggleIcon>
  );
};

export type FitViewControlProps = Button.IconProps;

export const FitViewControl = ({
  onClick,
  ...props
}: FitViewControlProps): ReactElement => {
  const { fitView } = useReactFlow();
  return (
    <Button.Icon
      onClick={(e) => {
        fitView();
        onClick?.(e);
      }}
      tooltip={<Text.Text level="small">Fit view to contents</Text.Text>}
      tooltipLocation={XYLocation.RIGHT_CENTER.crude}
      {...props}
    >
      <Icon.Expand />
    </Button.Icon>
  );
};

export const PID = (props: PIDProps): ReactElement => (
  <ReactFlowProvider>
    <Core {...props} />
  </ReactFlowProvider>
);
