// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ReactElement,
  createContext,
  useCallback,
  useMemo,
  useRef,
  useState,
  useContext as reactUseContext,
  type ComponentPropsWithoutRef,
  useEffect,
  memo,
} from "react";

import { Icon } from "@synnaxlabs/media";
import { box, location, xy } from "@synnaxlabs/x";
import ReactFlow, {
  ReactFlowProvider,
  type Viewport as RFViewport,
  useOnViewportChange as useRFOnViewportChange,
  applyEdgeChanges as rfApplyEdgeChanges,
  applyNodeChanges as rfApplyNodeChanges,
  addEdge as rfAddEdge,
  Background as RFBackground,
  type Edge as RFEdge,
  type NodeProps as RFNodeProps,
  type NodeChange as RFNodeChange,
  type EdgeChange as RFEdgeChange,
  type Connection as RFConnection,
  type ReactFlowProps,
  useReactFlow,
  ConnectionMode,
  updateEdge,
  type EdgeProps as RFEdgeProps,
  SelectionMode,
  type FitViewOptions,
  type ProOptions,
} from "reactflow";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useCombinedRefs } from "@/hooks";
import { useMemoCompare } from "@/memo";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";
import { type RenderProp } from "@/util/renderProp";
import { Viewport as CoreViewport } from "@/viewport";
import { Canvas } from "@/vis/canvas";
import { diagram } from "@/vis/diagram/aether";
import { Edge as EdgeComponent } from "@/vis/diagram/edge";
import { type connector } from "@/vis/diagram/edge/connector";
import { CustomConnectionLine } from "@/vis/diagram/edge/Edge";
import {
  type Edge,
  type Node,
  type Viewport,
  edgeConverter,
  nodeConverter,
  translateEdgesForward,
  translateNodesForward,
  translateViewportBackward,
  translateViewportForward,
} from "@/vis/diagram/types";

import "@/vis/diagram/Diagram.css";
import "reactflow/dist/style.css";

export interface SymbolProps {
  symbolKey: string;
  position: xy.XY;
  selected: boolean;
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
  initialViewport = { position: xy.ZERO, zoom: 1 },
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

const isValidConnection = (connection: RFConnection): boolean =>
  connection.source !== connection.target;

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
  zoomOnDoubleClick: false,
};

const NOT_EDITABLE_PROPS: ReactFlowProps = {
  connectionRadius: 0,
  nodesDraggable: false,
  nodesConnectable: false,
  elementsSelectable: false,
  panOnDrag: false,
  panOnScroll: false,
  zoomOnScroll: false,
  zoomOnDoubleClick: false,
  zoomOnPinch: false,
  edgesFocusable: false,
  edgesUpdatable: false,
  nodesFocusable: false,
  edgeUpdaterRadius: 0,
};

const FIT_VIEW_OPTIONS: FitViewOptions = {
  padding: 0,
  maxZoom: 1,
  minZoom: 0.5,
};

const PRO_OPTIONS: ProOptions = {
  hideAttribution: true,
};

export interface DiagramProps
  extends UseReturn,
    Omit<ComponentPropsWithoutRef<"div">, "onError"> {
  triggers?: CoreViewport.UseTriggers;
}

interface ContextValue {
  editable: boolean;
  onEditableChange: (v: boolean) => void;
  registerNodeRenderer: (renderer: RenderProp<SymbolProps>) => void;
}

const Context = createContext<ContextValue>({
  editable: true,
  onEditableChange: () => {},
  registerNodeRenderer: () => {},
});

export const useContext = (): ContextValue => reactUseContext(Context);

export interface NodeRendererProps {
  children: RenderProp<SymbolProps>;
}

export const NodeRenderer = memo(
  ({ children }: NodeRendererProps): ReactElement | null => {
    const { registerNodeRenderer } = useContext();
    useEffect(() => registerNodeRenderer(children), [registerNodeRenderer, children]);
    return null;
  },
);
NodeRenderer.displayName = "NodeRenderer";

const DELETE_KEY_CODES: Triggers.Trigger = ["Backspace", "Delete"];

const Core = Aether.wrap<DiagramProps>(
  diagram.Diagram.TYPE,
  // eslint-disable-next-line react/display-name
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
    triggers: pTriggers,
    onViewportChange,
    ...props
  }): ReactElement => {
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: diagram.Diagram.TYPE,
      schema: diagram.diagramStateZ,
      initialState: {
        position: viewport.position,
        region: box.ZERO,
        zoom: viewport.zoom,
      },
    });

    const defaultEdgeColor = Theming.use().colors.gray.l9.hex;

    const triggers = useMemoCompare(
      () => pTriggers ?? CoreViewport.DEFAULT_TRIGGERS.zoom,
      Triggers.compareModeConfigs,
      [pTriggers],
    );

    const { fitView } = useReactFlow();
    const resizeRef = Canvas.useRegion(
      useCallback(
        (b) => {
          setState((prev) => ({ ...prev, region: b }));
        },
        [setState],
      ),
    );

    // For some reason, react flow repeatedly calls onViewportChange with the same
    // parameters, so we do a need equality check to prevent unnecessary re-renders.
    const viewportRef = useRef<RFViewport | null>(null);
    const handleViewport = useCallback(
      (vp: RFViewport): void => {
        const prev = viewportRef.current;
        if (prev != null && prev.x === vp.x && prev.y === vp.y && prev.zoom === vp.zoom)
          return;
        viewportRef.current = vp;
        setState((prev) => ({ ...prev, position: vp, zoom: vp.zoom }));
        onViewportChange(translateViewportBackward(vp));
      },
      [setState, onViewportChange],
    );

    useRFOnViewportChange({
      onStart: handleViewport,
      onChange: handleViewport,
      onEnd: handleViewport,
    });

    const [renderer, setRenderer] = useState<RenderProp<SymbolProps>>(() => () => null);

    const registerNodeRenderer = useCallback(
      (renderer: RenderProp<SymbolProps>) => setRenderer(() => renderer),
      [],
    );

    const nodeTypes = useMemo(
      () => ({
        custom: ({ id, xPos, yPos, selected }: RFNodeProps) => {
          return renderer({
            symbolKey: id,
            position: { x: xPos, y: yPos },
            selected,
          });
        },
      }),
      [renderer],
    );

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
          nodeConverter(nodesRef.current, (n) => rfApplyNodeChanges(changes, n)),
        ),
      [onNodesChange],
    );

    const handleEdgesChange = useCallback(
      (changes: RFEdgeChange[]) =>
        onEdgesChange(
          edgeConverter(
            edgesRef.current,
            (e) => rfApplyEdgeChanges(changes, e),
            defaultEdgeColor,
          ),
        ),
      [onEdgesChange, defaultEdgeColor],
    );

    const handleEdgeUpdate = useCallback(
      (oldEdge: RFEdge, newConnection: RFConnection) =>
        onEdgesChange(
          edgeConverter(
            edgesRef.current,
            (e) => updateEdge(oldEdge, newConnection, e),
            defaultEdgeColor,
          ),
        ),
      [],
    );

    const handleConnect = useCallback(
      (conn: RFConnection) => {
        onEdgesChange(
          edgeConverter(edgesRef.current, (e) => rfAddEdge(conn, e), defaultEdgeColor),
        );
      },
      [onEdgesChange, defaultEdgeColor],
    );

    const handleEdgeSegmentsChange = useCallback(
      (id: string, segments: connector.Segment[]) => {
        const next = [...edgesRef.current];
        const index = next.findIndex((e) => e.key === id);
        if (index === -1) return;
        next[index] = { ...next[index], segments };
        edgesRef.current = next;
        onEdgesChange(next);
      },
      [onEdgesChange],
    );

    const editableProps = editable ? EDITABLE_PROPS : NOT_EDITABLE_PROPS;

    const EDGE_TYPES = useMemo(
      () => ({
        default: (props: RFEdgeProps) => (
          <EdgeComponent
            key={props.id}
            {...props}
            segments={props.data.segments}
            color={props.data.color}
            onSegmentsChange={useCallback(
              (segment) => handleEdgeSegmentsChange(props.id, segment),
              [props.id],
            )}
          />
        ),
      }),
      [handleEdgeSegmentsChange],
    );

    const triggerRef = useRef<HTMLElement>(null);
    Triggers.use({
      triggers: triggers.zoomReset,
      callback: useCallback(
        ({ stage, cursor }: Triggers.UseEvent) => {
          const reg = triggerRef.current;
          if (reg == null || stage !== "start" || !box.contains(reg, cursor)) return;
          fitView();
        },
        [fitView],
      ),
    });

    const selectTriggers = Triggers.purgeMouse(triggers.select)[0] ?? null;
    const panTriggers = Triggers.purgeMouse(triggers.pan)[0] ?? null;
    const zoomTriggers = Triggers.purgeMouse(triggers.zoom)[0] ?? null;
    const triggerProps: Partial<ReactFlowProps> = {
      selectionOnDrag: selectTriggers == null,
      panOnDrag: panTriggers == null,
      selectionKeyCode: selectTriggers,
      panActivationKeyCode: panTriggers,
      zoomActivationKeyCode: zoomTriggers,
    };

    const combinedRefs = useCombinedRefs(triggerRef, resizeRef);

    return (
      <Context.Provider value={{ editable, onEditableChange, registerNodeRenderer }}>
        <Aether.Composite path={path}>
          <ReactFlow
            {...triggerProps}
            className={CSS(CSS.B("diagram"), CSS.editable(editable))}
            nodes={nodes_}
            edges={edges_}
            nodeTypes={nodeTypes}
            edgeTypes={EDGE_TYPES}
            ref={combinedRefs}
            fitView
            onNodesChange={handleNodesChange}
            onEdgesChange={handleEdgesChange}
            onConnect={handleConnect}
            onEdgeUpdate={handleEdgeUpdate}
            defaultViewport={translateViewportForward(viewport)}
            connectionLineComponent={CustomConnectionLine}
            elevateEdgesOnSelect
            minZoom={0.5}
            maxZoom={1.2}
            isValidConnection={isValidConnection}
            connectionMode={ConnectionMode.Loose}
            snapGrid={[3, 3]}
            fitViewOptions={FIT_VIEW_OPTIONS}
            selectionMode={SelectionMode.Partial}
            proOptions={PRO_OPTIONS}
            deleteKeyCode={DELETE_KEY_CODES}
            {...props}
            style={{
              [CSS.var("diagram-zoom")]: viewport.zoom,
              ...props.style,
            }}
            {...editableProps}
          >
            {children}
          </ReactFlow>
        </Aether.Composite>
      </Context.Provider>
    );
  },
);

export const Background = (): ReactElement | null => {
  const { editable } = useContext();
  return editable ? <RFBackground /> : null;
};

export interface ControlsProps extends Align.PackProps {}

export const Controls = ({ children, ...props }: ControlsProps): ReactElement => (
  <Align.Pack direction="y" className={CSS.BE("diagram", "controls")} {...props}>
    {children}
  </Align.Pack>
);

export type ToggleEditControlProps = Omit<
  Button.ToggleIconProps,
  "value" | "onChange" | "children"
>;

export const ToggleEditControl = ({
  onClick,
  ...props
}: ToggleEditControlProps): ReactElement => {
  const { editable, onEditableChange } = useContext();
  return (
    <Button.ToggleIcon
      onChange={() => onEditableChange(!editable)}
      value={editable}
      uncheckedVariant="outlined"
      checkedVariant="filled"
      tooltipLocation={location.RIGHT_CENTER}
      tooltip={
        editable ? (
          <Text.Text level="small">Disable edit mode</Text.Text>
        ) : (
          <Text.Text level="small">Enable edit mode</Text.Text>
        )
      }
      {...props}
    >
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
    </Button.ToggleIcon>
  );
};

export type FitViewControlProps = Omit<Button.IconProps, "children">;

export const FitViewControl = ({
  onClick,
  ...props
}: FitViewControlProps): ReactElement => {
  const { fitView } = useReactFlow();
  return (
    <Button.Icon
      onClick={(e) => {
        fitView(FIT_VIEW_OPTIONS);
        onClick?.(e);
      }}
      tooltip={<Text.Text level="small">Fit view to contents</Text.Text>}
      tooltipLocation={location.RIGHT_CENTER}
      variant="outlined"
      {...props}
    >
      <Icon.Expand />
    </Button.Icon>
  );
};

export const Diagram = (props: DiagramProps): ReactElement => (
  <ReactFlowProvider>
    <Core {...props} />
  </ReactFlowProvider>
);
