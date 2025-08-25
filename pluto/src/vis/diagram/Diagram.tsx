// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/diagram/Diagram.css";
import "@xyflow/react/dist/base.css";

import { box, color, location, xy } from "@synnaxlabs/x";
import {
  addEdge as rfAddEdge,
  applyEdgeChanges as rfApplyEdgeChanges,
  applyNodeChanges as rfApplyNodeChanges,
  Background as RFBackground,
  type Connection as RFConnection,
  ConnectionMode,
  type Edge as RFEdge,
  type EdgeChange as RFEdgeChange,
  type EdgeProps as RFEdgeProps,
  type FitViewOptions,
  type IsValidConnection,
  type Node as RFNode,
  type NodeChange,
  type NodeProps as RFNodeProps,
  type ProOptions,
  ReactFlow,
  type ReactFlowProps,
  ReactFlowProvider,
  reconnectEdge,
  SelectionMode,
  useOnViewportChange as useRFOnViewportChange,
  useReactFlow,
  type Viewport as RFViewport,
} from "@xyflow/react";
import {
  type ComponentPropsWithoutRef,
  createContext,
  memo,
  type ReactElement,
  use as reactUse,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useCombinedRefs, useDebouncedCallback, useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { useMemoCompare, useMemoDeepEqual } from "@/memo";
import { Select } from "@/select";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";
import { Viewport as CoreViewport } from "@/viewport";
import { Canvas } from "@/vis/canvas";
import { diagram } from "@/vis/diagram/aether";
import {
  type Edge,
  edgeConverter,
  type Node,
  nodeConverter,
  type RFEdgeData,
  translateEdgesForward,
  translateNodesForward,
  translateViewportBackward,
  translateViewportForward,
  type Viewport,
} from "@/vis/diagram/aether/types";
import { Edge as EdgeComponent } from "@/vis/diagram/edge";
import { type connector } from "@/vis/diagram/edge/connector";
import { CustomConnectionLine } from "@/vis/diagram/edge/Edge";
import { type PathType } from "@/vis/diagram/edge/paths";

export interface SymbolProps {
  symbolKey: string;
  position: xy.XY;
  selected: boolean;
  draggable: boolean;
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
  const [viewportMode, onViewportModeChange] = useState<CoreViewport.Mode>("select");
  const [nodes, onNodesChange] = useState<Node[]>(initialNodes);
  const [edges, onEdgesChange] = useState<Edge[]>(initialEdges);
  const [viewport, onViewportChange] = useState<Viewport>(initialViewport);
  const [fitViewOnResize, setFitViewOnResize] = useState(false);

  return {
    viewport,
    onViewportChange,
    edges,
    nodes,
    onNodesChange,
    onEdgesChange,
    editable,
    onEditableChange,
    fitViewOnResize,
    setFitViewOnResize,
    viewportMode,
    onViewportModeChange,
  };
};

const isValidConnection: IsValidConnection = (): boolean => true;

export interface UseReturn {
  edges: Edge[];
  nodes: Node[];
  onNodesChange: (nodes: Node[], changes: NodeChange[]) => void;
  onEdgesChange: (edges: Edge[]) => void;
  editable: boolean;
  onEditableChange: (v: boolean) => void;
  onViewportChange: (vp: Viewport) => void;
  viewport: Viewport;
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
  viewportMode: CoreViewport.Mode;
  onViewportModeChange: (v: CoreViewport.Mode) => void;
}

const EDITABLE_PROPS: ReactFlowProps = {
  nodesDraggable: true,
  nodesConnectable: true,
  elementsSelectable: true,
  zoomOnDoubleClick: false,
  nodeClickDistance: 5,
  reconnectRadius: 15,
  connectionRadius: 30,
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
  edgesReconnectable: false,
  nodesFocusable: false,
  reconnectRadius: 0,
};

const FIT_VIEW_OPTIONS: FitViewOptions = {
  maxZoom: 1,
  minZoom: 0.5,
  padding: 0.05,
};

const PRO_OPTIONS: ProOptions = {
  hideAttribution: true,
};

export interface DiagramProps
  extends UseReturn,
    Omit<ComponentPropsWithoutRef<"div">, "onError">,
    Pick<z.infer<typeof diagram.Diagram.stateZ>, "visible">,
    Aether.ComponentProps {
  triggers?: CoreViewport.UseTriggers;
  dragHandleSelector?: string;
}

interface ContextValue {
  editable: boolean;
  visible: boolean;
  onEditableChange: (v: boolean) => void;
  viewportMode: CoreViewport.Mode;
  onViewportModeChange: (v: CoreViewport.Mode) => void;
  registerNodeRenderer: (renderer: RenderProp<SymbolProps>) => void;
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
}

const Context = createContext<ContextValue>({
  editable: true,
  visible: true,
  viewportMode: "select",
  onViewportModeChange: () => {},
  onEditableChange: () => {},
  registerNodeRenderer: () => {},
  fitViewOnResize: false,
  setFitViewOnResize: () => {},
});

export const useContext = () => reactUse(Context);

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

const PAN_PROPS: Partial<ReactFlowProps> = {
  panOnDrag: [Triggers.MOUSE_LEFT_NUMBER, Triggers.MOUSE_MIDDLE_NUMBER],
};

const SELECT_PROPS: Partial<ReactFlowProps> = {
  selectionOnDrag: true,
  panOnDrag: [Triggers.MOUSE_MIDDLE_NUMBER],
};

const viewPortModeToRFProps = (mode: CoreViewport.Mode): Partial<ReactFlowProps> => {
  if (mode === "pan") return PAN_PROPS;
  return SELECT_PROPS;
};

const Core = ({
  aetherKey,
  onNodesChange,
  onEdgesChange,
  nodes,
  edges,
  onEditableChange,
  editable,
  viewport,
  triggers: pTriggers,
  onViewportChange,
  fitViewOnResize,
  setFitViewOnResize,
  visible,
  dragHandleSelector,
  viewportMode,
  onViewportModeChange,
  ...rest
}: DiagramProps): ReactElement => {
  const memoProps = useMemoDeepEqual({ visible });
  const [{ path }, , setState] = Aether.use({
    aetherKey,
    type: diagram.Diagram.TYPE,
    schema: diagram.diagramStateZ,
    initialState: {
      position: viewport.position,
      region: box.ZERO,
      zoom: viewport.zoom,
      ...memoProps,
    },
  });
  const { fitView } = useReactFlow();
  const debouncedFitView = useDebouncedCallback((args) => void fitView(args), 50, [
    fitView,
  ]);

  const resizeRef = Canvas.useRegion(
    useCallback(
      (b) => {
        if (fitViewOnResize) debouncedFitView(FIT_VIEW_OPTIONS);
        setState((prev) => ({ ...prev, region: b }));
      },
      [setState, debouncedFitView, fitViewOnResize],
    ),
  );
  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);

  const defaultEdgeColor = color.hex(Theming.use().colors.gray.l11);

  const triggers = useMemoCompare(
    () => pTriggers ?? CoreViewport.DEFAULT_TRIGGERS.zoom,
    Triggers.compareModeConfigs,
    [pTriggers],
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
      if (isNaN(vp.x) || isNaN(vp.y) || isNaN(vp.zoom)) return;
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
      custom: ({
        id,
        positionAbsoluteX: x,
        positionAbsoluteY: y,
        selected = false,
        draggable = true,
      }: RFNodeProps) =>
        renderer({ symbolKey: id, position: { x, y }, selected, draggable }),
    }),
    [renderer],
  );

  const edgesRef = useRef(edges);
  const edges_ = useMemo<RFEdge<RFEdgeData>[]>(() => {
    edgesRef.current = edges;
    return translateEdgesForward(edges);
  }, [edges]);
  const nodesRef = useRef(nodes);
  const nodes_ = useMemo(() => {
    nodesRef.current = nodes;
    return translateNodesForward(nodes, dragHandleSelector);
  }, [nodes, dragHandleSelector]);

  const handleNodesChange = useCallback(
    (changes: NodeChange[]) =>
      onNodesChange(
        nodeConverter(nodesRef.current, (n) => rfApplyNodeChanges(changes, n)),
        changes,
      ),
    [onNodesChange],
  );

  const handleEdgesChange = useCallback(
    (changes: RFEdgeChange<RFEdge<RFEdgeData>>[]) =>
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
    (oldEdge: RFEdge<RFEdgeData>, newConnection: RFConnection) =>
      onEdgesChange(
        edgeConverter(
          edgesRef.current,
          (e) => reconnectEdge(oldEdge, newConnection, e),
          defaultEdgeColor,
        ),
      ),
    [],
  );

  const handleConnect = useCallback(
    (conn: RFConnection) =>
      onEdgesChange(
        edgeConverter(edgesRef.current, (e) => rfAddEdge(conn, e), defaultEdgeColor),
      ),
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

  const handleEdgeSegmentsChangeRef = useSyncedRef(handleEdgeSegmentsChange);

  const edgeTypes = useMemo(
    () => ({
      default: (props: RFEdgeProps<RFEdge<RFEdgeData>>) => (
        <EdgeComponent
          key={props.id}
          {...props}
          segments={props.data?.segments ?? []}
          color={props.data?.color}
          variant={props.data?.variant as PathType}
          onSegmentsChange={useCallback(
            (segment) => handleEdgeSegmentsChangeRef.current(props.id, segment),
            [props.id],
          )}
        />
      ),
    }),
    [],
  );

  const triggerRef = useRef<HTMLElement>(null);
  Triggers.use({
    triggers: triggers.zoomReset,
    callback: useCallback(
      ({ stage, cursor }: Triggers.UseEvent) => {
        const reg = triggerRef.current;
        if (reg == null || stage !== "start" || !box.contains(reg, cursor)) return;
        void fitView();
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

  const ctxValue = useMemo(
    () => ({
      visible,
      editable,
      onEditableChange,
      registerNodeRenderer,
      fitViewOnResize,
      setFitViewOnResize,
      viewportMode,
      onViewportModeChange,
    }),
    [
      editable,
      visible,
      onEditableChange,
      registerNodeRenderer,
      fitViewOnResize,
      viewportMode,
      onViewportModeChange,
    ],
  );

  return (
    <Context value={ctxValue}>
      <Aether.Composite path={path}>
        {visible && (
          <ReactFlow<RFNode, RFEdge<RFEdgeData>>
            {...triggerProps}
            className={CSS(
              CSS.B("diagram"),
              CSS.editable(editable),
              CSS.BE("symbol", "container"),
            )}
            nodes={nodes_}
            // @ts-expect-error - edge types
            edges={edges_}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            ref={combinedRefs}
            fitView
            onNodesChange={handleNodesChange}
            onEdgesChange={handleEdgesChange}
            onConnect={handleConnect}
            onReconnect={handleEdgeUpdate}
            defaultViewport={translateViewportForward(viewport)}
            connectionLineComponent={CustomConnectionLine}
            elevateEdgesOnSelect
            minZoom={0.5}
            maxZoom={1.2}
            isValidConnection={isValidConnection}
            connectionMode={ConnectionMode.Loose}
            fitViewOptions={FIT_VIEW_OPTIONS}
            selectionMode={SelectionMode.Partial}
            proOptions={PRO_OPTIONS}
            deleteKeyCode={DELETE_KEY_CODES}
            {...rest}
            style={{ [CSS.var("diagram-zoom")]: viewport.zoom, ...rest.style }}
            {...editableProps}
            {...viewPortModeToRFProps(viewportMode)}
            nodesDraggable={editable}
          />
        )}
      </Aether.Composite>
    </Context>
  );
};

export const Background = (): ReactElement | null => {
  const { editable } = useContext();
  return editable ? <RFBackground /> : null;
};

export interface ControlsProps extends Flex.BoxProps {}

export const Controls = (props: ControlsProps): ReactElement => (
  <Flex.Box x className={CSS.BE("diagram", "controls")} {...props} />
);

export interface ToggleEditControlProps
  extends Omit<Button.ToggleProps, "value" | "onChange" | "children"> {}

const CONTROL_TOOLTIP_LOCATION = location.BOTTOM_LEFT;

export const ToggleEditControl = ({
  onClick,
  ...rest
}: ToggleEditControlProps): ReactElement => {
  const { editable, onEditableChange } = useContext();
  return (
    <Button.Toggle
      onChange={() => onEditableChange(!editable)}
      value={editable}
      uncheckedVariant="outlined"
      checkedVariant="filled"
      tooltipLocation={CONTROL_TOOLTIP_LOCATION}
      size="small"
      tooltip={`${editable ? "Disable" : "Enable"} editing`}
      {...rest}
    >
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
    </Button.Toggle>
  );
};

export interface FitViewControlProps
  extends Omit<Button.ToggleProps, "children" | "onChange" | "value"> {}

export const FitViewControl = ({
  onClick,
  ...rest
}: FitViewControlProps): ReactElement => {
  const { fitView } = useReactFlow();
  const { fitViewOnResize, setFitViewOnResize } = useContext();
  return (
    <Button.Toggle
      onClick={(e) => {
        void fitView(FIT_VIEW_OPTIONS);
        onClick?.(e);
      }}
      value={fitViewOnResize}
      onChange={setFitViewOnResize}
      rightClickToggle
      tooltip={<Text.Text level="small">Fit view to contents</Text.Text>}
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      {...rest}
    >
      <Icon.Expand />
    </Button.Toggle>
  );
};

export const VIEWPORT_MODES = ["zoom", "pan", "select"] as const;
const PAN_TRIGGER: Triggers.Trigger[] = [["MouseMiddle"]];
const SELECT_TRIGGER: Triggers.Trigger[] = [["MouseLeft"]];

export const SelectViewportModeControl = (): ReactElement => {
  const { viewportMode, onViewportModeChange } = useContext();
  return (
    <Select.Buttons
      keys={VIEWPORT_MODES}
      value={viewportMode}
      onChange={onViewportModeChange}
    >
      <Select.Button
        itemKey="pan"
        size="small"
        tooltip={<CoreViewport.TooltipText mode="pan" triggers={PAN_TRIGGER} />}
        tooltipLocation={CONTROL_TOOLTIP_LOCATION}
      >
        <Icon.Pan />
      </Select.Button>
      <Select.Button
        itemKey="select"
        size="small"
        tooltip={<CoreViewport.TooltipText mode="select" triggers={SELECT_TRIGGER} />}
        tooltipLocation={CONTROL_TOOLTIP_LOCATION}
      >
        <Icon.Selection />
      </Select.Button>
    </Select.Buttons>
  );
};

export const Diagram = (props: DiagramProps): ReactElement => (
  <ReactFlowProvider>
    <Core {...props} />
  </ReactFlowProvider>
);
