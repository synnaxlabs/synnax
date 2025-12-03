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

import { box, color, location, type record, xy } from "@synnaxlabs/x";
import {
  addEdge as rfAddEdge,
  applyEdgeChanges as rfApplyEdgeChanges,
  applyNodeChanges as rfApplyNodeChanges,
  Background as RFBackground,
  type Connection as RFConnection,
  type ConnectionLineComponent,
  ConnectionMode,
  type Edge as RFEdge,
  type EdgeChange as RFEdgeChange,
  type EdgeProps as RFEdgeProps,
  type FitViewOptions as RFFitViewOptions,
  type IsValidConnection,
  type Node as RFNode,
  type NodeChange,
  type NodeProps as RFNodeProps,
  type ProOptions,
  ReactFlow,
  type ReactFlowInstance,
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
  memo,
  type ReactElement,
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
import { context } from "@/context";
import { CSS } from "@/css";
import { useCombinedRefs, useDebouncedCallback } from "@/hooks";
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
  translateEdgesForward,
  translateNodesForward,
  translateViewportBackward,
  translateViewportForward,
  type Viewport,
} from "@/vis/diagram/aether/types";

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
    Pick<z.infer<typeof diagram.Diagram.stateZ>, "visible" | "autoRenderInterval">,
    Aether.ComponentProps,
    Pick<
      ReactFlowProps,
      "minZoom" | "maxZoom" | "fitViewOptions" | "snapGrid" | "snapToGrid"
    > {
  triggers?: CoreViewport.UseTriggers;
  dragHandleSelector?: string;
}

interface ContextValue {
  editable: boolean;
  visible: boolean;
  onEditableChange: (v: boolean) => void;
  registerNodeRenderer: (renderer: RenderProp<SymbolProps>) => void;
  registerEdgeRenderer: (renderer: RenderProp<EdgeProps<record.Unknown>>) => void;
  viewportMode: CoreViewport.Mode;
  onViewportModeChange: (v: CoreViewport.Mode) => void;
  registerConnectionLineComponent: (component: ConnectionLineComponent<RFNode>) => void;
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
  fitViewOptions: FitViewOptions;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    editable: true,
    fitViewOnResize: false,
    fitViewOptions: FIT_VIEW_OPTIONS,
    onEditableChange: () => {},
    onViewportModeChange: () => {},
    registerConnectionLineComponent: () => {},
    registerEdgeRenderer: () => {},
    registerNodeRenderer: () => {},
    setFitViewOnResize: () => {},
    viewportMode: "select",
    visible: true,
  },
  displayName: "Diagram.Context",
});
export { useContext };

export interface NodeRendererProps {
  children: RenderProp<SymbolProps>;
}

export interface EdgeProps<D extends record.Unknown> extends RFEdgeProps<RFEdge<D>> {
  onDataChange: (data: D) => void;
}

export const NodeRenderer = memo(
  ({ children }: NodeRendererProps): ReactElement | null => {
    const { registerNodeRenderer } = useContext();
    useEffect(() => registerNodeRenderer(children), [registerNodeRenderer, children]);
    return null;
  },
);
NodeRenderer.displayName = "NodeRenderer";

export interface EdgeRendererProps<D extends record.Unknown> {
  connectionLineComponent: ConnectionLineComponent<RFNode>;
  children: RenderProp<EdgeProps<D>>;
}

const CoreEdgeRenderer = memo(
  <D extends record.Unknown>({
    children,
    connectionLineComponent,
  }: EdgeRendererProps<D>): ReactElement | null => {
    const { registerEdgeRenderer, registerConnectionLineComponent } = useContext();
    useEffect(
      () => registerEdgeRenderer(children as RenderProp<EdgeProps<record.Unknown>>),
      [registerEdgeRenderer, children],
    );
    useEffect(
      () => registerConnectionLineComponent(connectionLineComponent),
      [registerConnectionLineComponent, connectionLineComponent],
    );
    return null;
  },
);
CoreEdgeRenderer.displayName = "EdgeRenderer";
export const EdgeRenderer = CoreEdgeRenderer as <D extends record.Unknown>(
  props: EdgeRendererProps<D>,
) => ReactElement | null;

export type FitViewOptions = RFFitViewOptions;

const DELETE_KEY_CODES: Triggers.Trigger = ["Backspace", "Delete"];

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
  fitViewOptions = FIT_VIEW_OPTIONS,
  className,
  dragHandleSelector,
  snapGrid,
  snapToGrid = false,
  viewportMode,
  onViewportModeChange,
  autoRenderInterval,
  ...rest
}: DiagramProps): ReactElement => {
  const memoProps = useMemoDeepEqual({ visible, autoRenderInterval });
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
        if (fitViewOnResize) debouncedFitView(fitViewOptions);
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

  const [nodeRenderer, setNodeRenderer] = useState<RenderProp<SymbolProps>>(
    () => () => null,
  );
  const [edgeRenderer, setEdgeRenderer] = useState<RenderProp<
    EdgeProps<record.Unknown>
  > | null>(null);
  const [connectionLineComponent, setConnectionLineComponent] = useState<
    ConnectionLineComponent<RFNode> | undefined
  >(undefined);

  const registerNodeRenderer = useCallback(
    (renderer: RenderProp<SymbolProps>) => setNodeRenderer(() => renderer),
    [setNodeRenderer],
  );

  const registerEdgeRenderer = useCallback(
    (renderer: RenderProp<EdgeProps<record.Unknown>>) =>
      setEdgeRenderer(() => renderer),
    [setEdgeRenderer],
  );

  const registerConnectionLineComponent = useCallback(
    (component: ConnectionLineComponent<RFNode>) =>
      setConnectionLineComponent(() => component),
    [setConnectionLineComponent],
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
        nodeRenderer({ symbolKey: id, position: { x, y }, selected, draggable }),
    }),
    [nodeRenderer],
  );

  const handleDataChange = useCallback(
    (id: string, data: record.Unknown) => {
      const next = [...edgesRef.current];
      const index = next.findIndex((e) => e.key === id);
      if (index === -1) return;
      next[index] = { ...next[index], data };
      edgesRef.current = next;
      onEdgesChange(next);
    },
    [onEdgesChange, defaultEdgeColor],
  );

  const edgeTypes = useMemo(() => {
    if (edgeRenderer == null) return undefined;
    return {
      default: (props: RFEdgeProps<RFEdge<record.Unknown>>) =>
        edgeRenderer({
          ...props,
          onDataChange: (data) => handleDataChange(props.id, data),
        }),
    };
  }, [edgeRenderer, handleDataChange]);

  const edgesRef = useRef(edges);
  const edges_ = useMemo<RFEdge<record.Unknown>[]>(() => {
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
    (changes: RFEdgeChange<RFEdge<record.Unknown>>[]) =>
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
    (oldEdge: RFEdge<record.Unknown>, newConnection: RFConnection) =>
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

  const editableProps = editable ? EDITABLE_PROPS : NOT_EDITABLE_PROPS;

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

  const handleInit = useCallback(
    (i: ReactFlowInstance<RFNode, RFEdge<record.Unknown>>) => {
      void i.fitView(fitViewOptions);
    },
    [fitViewOptions],
  );

  const ctxValue = useMemo(
    () => ({
      visible,
      editable,
      onEditableChange,
      registerNodeRenderer,
      registerEdgeRenderer,
      registerConnectionLineComponent,
      fitViewOnResize,
      setFitViewOnResize,
      fitViewOptions,
      viewportMode,
      onViewportModeChange,
    }),
    [
      editable,
      visible,
      registerConnectionLineComponent,
      onEditableChange,
      registerNodeRenderer,
      registerEdgeRenderer,
      fitViewOnResize,
      fitViewOptions,
      viewportMode,
      onViewportModeChange,
    ],
  );

  const defaultEdgeOptions = useMemo(
    () => ({
      type: edgeRenderer == null ? "smoothstep" : "default",
    }),
    [edgeRenderer],
  );

  const style = useMemo(
    () => ({ [CSS.var("diagram-zoom")]: viewport.zoom, ...rest.style }),
    [viewport.zoom, rest.style],
  );

  return (
    <Context value={ctxValue}>
      <Aether.Composite path={path}>
        {visible && (
          <ReactFlow<RFNode, RFEdge<record.Unknown>>
            {...triggerProps}
            className={CSS(
              className,
              CSS.B("diagram"),
              CSS.editable(editable),
              CSS.BE("symbol", "container"),
            )}
            nodes={nodes_}
            edges={edges_}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            ref={combinedRefs}
            fitView
            onNodesChange={handleNodesChange}
            onEdgesChange={handleEdgesChange}
            onConnect={handleConnect}
            connectionLineComponent={connectionLineComponent}
            onReconnect={handleEdgeUpdate}
            defaultViewport={translateViewportForward(viewport)}
            elevateEdgesOnSelect
            defaultEdgeOptions={defaultEdgeOptions}
            minZoom={fitViewOptions.minZoom}
            maxZoom={fitViewOptions.maxZoom}
            isValidConnection={isValidConnection}
            connectionMode={ConnectionMode.Loose}
            fitViewOptions={fitViewOptions}
            selectionMode={SelectionMode.Partial}
            proOptions={PRO_OPTIONS}
            deleteKeyCode={DELETE_KEY_CODES}
            snapGrid={snapGrid}
            snapToGrid={snapToGrid}
            {...rest}
            style={style}
            {...editableProps}
            nodesDraggable={editable}
            onInit={handleInit}
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

export interface ToggleEditControlProps
  extends Omit<Button.ToggleProps, "value" | "onChange" | "children"> {}

export const ToggleEditControl = ({
  onClick,
  ...rest
}: ToggleEditControlProps): ReactElement => {
  const { editable, onEditableChange } = useContext();
  return (
    <Button.Toggle
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      tooltip={
        <Text.Text level="small">{`${editable ? "Disable" : "Enable"} editing`}</Text.Text>
      }
      {...rest}
      onChange={() => onEditableChange(!editable)}
      value={editable}
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
      tooltip={<Text.Text level="small">Fit view to contents</Text.Text>}
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      {...rest}
      value={fitViewOnResize}
      onChange={setFitViewOnResize}
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
        tooltipLocation={location.BOTTOM_LEFT}
      >
        <Icon.Pan />
      </Select.Button>
      <Select.Button
        itemKey="select"
        size="small"
        tooltip={<CoreViewport.TooltipText mode="select" triggers={SELECT_TRIGGER} />}
        tooltipLocation={location.BOTTOM_LEFT}
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
