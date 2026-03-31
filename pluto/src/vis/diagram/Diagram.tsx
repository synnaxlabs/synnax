// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/diagram/Diagram.css";
import "@xyflow/react/dist/base.css";

import { box, type xy } from "@synnaxlabs/x";
import {
  type Connection as RFConnection,
  type ConnectionLineComponentProps as RFConnectionLineProps,
  ConnectionMode,
  type EdgeChange as RFEdgeChange,
  type EdgeProps as RFEdgeProps,
  type IsValidConnection,
  type NodeChange as RFNodeChange,
  type NodeProps as RFNodeProps,
  type ProOptions,
  ReactFlow,
  type ReactFlowInstance,
  type ReactFlowProps,
  ReactFlowProvider,
  SelectionMode,
  useOnViewportChange as useRFOnViewportChange,
  useReactFlow,
  type Viewport as RFViewport,
} from "@xyflow/react";
import {
  type ComponentPropsWithoutRef,
  type FC,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { useCombinedRefs, useDebouncedCallback } from "@/hooks";
import { useMemoCompare, useMemoDeepEqual } from "@/memo";
import { Triggers } from "@/triggers";
import { Viewport as BaseViewport } from "@/viewport";
import { Canvas } from "@/vis/canvas";
import { diagram } from "@/vis/diagram/aether";
import {
  type Edge,
  type EdgeChange,
  type Node,
  type NodeChange,
  translateEdgeChangeForward,
  translateEdgesForward,
  translateNodeChangeForward,
  translateNodesForward,
  translateViewportBackward,
  translateViewportForward,
  type Viewport,
} from "@/vis/diagram/aether/types";
import { Context } from "@/vis/diagram/Context";

export interface NodeProps {
  nodeKey: string;
  position: xy.XY;
  selected: boolean;
  draggable: boolean;
}

export interface RendererConfig {
  node: RenderProp<NodeProps, ReactElement>;
  edge?: RenderProp<diagram.EdgeProps, ReactElement>;
  connectionLine?: RenderProp<diagram.ConnectionLineProps, ReactElement>;
}

const isValidConnection: IsValidConnection = (): boolean => true;

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

const PRO_OPTIONS: ProOptions = {
  hideAttribution: true,
};

export interface DiagramProps
  extends
    Omit<ComponentPropsWithoutRef<"div">, "onError">,
    Pick<z.infer<typeof diagram.Diagram.stateZ>, "visible" | "autoRenderInterval">,
    Aether.ComponentProps,
    Pick<
      ReactFlowProps,
      "minZoom" | "maxZoom" | "fitViewOptions" | "snapGrid" | "snapToGrid"
    > {
  edges: Edge[];
  nodes: Node[];
  onNodesChange: (changes: NodeChange[]) => void;
  onEdgesChange: (changes: EdgeChange[]) => void;
  selected?: string[];
  onSelectionChange?: (selected: string[]) => void;
  editable: boolean;
  onEditableChange: (v: boolean) => void;
  onViewportChange: (vp: Viewport) => void;
  viewport: Viewport;
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
  viewportMode: BaseViewport.Mode;
  onViewportModeChange: (v: BaseViewport.Mode) => void;
  triggers?: BaseViewport.UseTriggers;
  dragHandleSelector?: string;
}

const DELETE_KEY_CODES: Triggers.Trigger = ["Backspace", "Delete"];

export const create = ({
  node: nodeRenderer,
  edge: edgeRenderer,
  connectionLine: connectionLineRenderer,
}: RendererConfig): FC<DiagramProps> => {
  const NodeWrapper = ({
    id,
    positionAbsoluteX: x,
    positionAbsoluteY: y,
    selected = false,
    draggable = true,
  }: RFNodeProps): ReactElement => {
    const position = useMemo(() => ({ x, y }), [x, y]);
    return nodeRenderer({ nodeKey: id, position, selected, draggable });
  };

  const nodeTypes = { custom: NodeWrapper };

  const EdgeWrapper = ({ id, sourceX, sourceY, sourcePosition, targetX, targetY, targetPosition, source, target, selected = false }: RFEdgeProps): ReactElement => {
    const s = useMemo(
      () => diagram.createEndpoint(sourceX, sourceY, sourcePosition),
      [sourceX, sourceY, sourcePosition],
    );
    const t = useMemo(
      () => diagram.createEndpoint(targetX, targetY, targetPosition),
      [targetX, targetY, targetPosition],
    );
    return edgeRenderer!({
      edgeKey: id,
      source: s,
      target: t,
      sourceNode: source,
      targetNode: target,
      selected,
    });
  };

  const edgeTypes = edgeRenderer != null ? { default: EdgeWrapper } : undefined;

  const ConnectionLine =
    connectionLineRenderer != null
      ? (rf: RFConnectionLineProps): ReactElement =>
          connectionLineRenderer({
            source: diagram.createEndpoint(rf.fromX, rf.fromY, rf.fromPosition),
            target: diagram.createEndpoint(rf.toX, rf.toY, rf.toPosition),
            status: rf.connectionStatus,
            style: rf.connectionLineStyle ?? {},
          })
      : undefined;

  const defaultEdgeOptions = {
    type: edgeRenderer != null ? "default" : "smoothstep",
  };

  const Base = ({
    aetherKey,
    onNodesChange,
    onEdgesChange,
    nodes,
    edges,
    selected,
    onSelectionChange,
    onEditableChange,
    editable,
    viewport,
    triggers: pTriggers,
    onViewportChange,
    fitViewOnResize,
    setFitViewOnResize,
    visible,
    fitViewOptions = diagram.FIT_VIEW_OPTIONS,
    className,
    dragHandleSelector,
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
        (region) => {
          if (fitViewOnResize) debouncedFitView(fitViewOptions);
          setState((prev) => ({ ...prev, region }));
        },
        [setState, debouncedFitView, fitViewOnResize],
      ),
    );
    useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);

    const triggers = useMemoCompare(
      () => pTriggers ?? BaseViewport.DEFAULT_TRIGGERS.zoom,
      Triggers.compareModeConfigs,
      [pTriggers],
    );

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

    const selectedSet = useMemo(() => new Set(selected), [selected]);

    const rfEdges = useMemo(
      () => translateEdgesForward(edges, selectedSet),
      [edges, selectedSet],
    );
    const rfNodes = useMemo(
      () => translateNodesForward(nodes, selectedSet, dragHandleSelector),
      [nodes, selectedSet, dragHandleSelector],
    );

    const selectedRef = useRef(selected);
    selectedRef.current = selected;

    const handleNodesChange = useCallback(
      (changes: RFNodeChange[]) => {
        const selChanges: Array<{ key: string; selected: boolean }> = [];
        const mutations: NodeChange[] = [];
        for (const change of changes) {
          const translated = translateNodeChangeForward(change);
          if (translated == null) continue;
          if (translated.type === "select") selChanges.push(translated);
          else mutations.push(translated);
        }
        if (selChanges.length > 0) {
          const next = new Set(selectedRef.current);
          for (const c of selChanges)
            if (c.selected) next.add(c.key);
            else next.delete(c.key);
          onSelectionChange?.([...next]);
        }
        if (mutations.length > 0) onNodesChange(mutations);
      },
      [onNodesChange, onSelectionChange],
    );

    const handleEdgesChange = useCallback(
      (changes: RFEdgeChange[]) => {
        const selChanges: Array<{ key: string; selected: boolean }> = [];
        const mutations: EdgeChange[] = [];
        for (const change of changes) {
          const translated = translateEdgeChangeForward(change);
          if (translated == null) continue;
          if (translated.type === "select") selChanges.push(translated);
          else mutations.push(translated);
        }
        if (selChanges.length > 0) {
          const next = new Set(selectedRef.current);
          for (const c of selChanges)
            if (c.selected) next.add(c.key);
            else next.delete(c.key);
          onSelectionChange?.([...next]);
        }
        if (mutations.length > 0) onEdgesChange(mutations);
      },
      [onEdgesChange, onSelectionChange],
    );

    const handleConnect = useCallback(
      (conn: RFConnection) => {
        const key = `${conn.source}-${conn.sourceHandle ?? ""}-${conn.target}-${conn.targetHandle ?? ""}`;
        const edge: Edge = {
          key,
          source: { node: conn.source, param: conn.sourceHandle ?? "" },
          target: { node: conn.target, param: conn.targetHandle ?? "" },
        };
        onEdgesChange([{ type: "add", edge }]);
      },
      [onEdgesChange],
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
      (i: ReactFlowInstance) => {
        void i.fitView(fitViewOptions);
      },
      [fitViewOptions],
    );

    const ctxValue = useMemo(
      () => ({
        visible,
        editable,
        onEditableChange,
        fitViewOnResize,
        setFitViewOnResize,
        fitViewOptions,
        viewportMode,
        onViewportModeChange,
      }),
      [
        editable,
        visible,
        onEditableChange,
        fitViewOnResize,
        fitViewOptions,
        viewportMode,
        onViewportModeChange,
      ],
    );

    const style = useMemo(
      () => ({ [CSS.var("diagram-zoom")]: viewport.zoom, ...rest.style }),
      [viewport.zoom, rest.style],
    );

    return (
      <Context value={ctxValue}>
        <Aether.Composite path={path}>
          {visible && (
            <ReactFlow
              {...triggerProps}
              className={CSS(
                className,
                CSS.B("diagram"),
                CSS.editable(editable),
                CSS.BE("symbol", "container"),
              )}
              nodes={rfNodes}
              edges={rfEdges}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              ref={combinedRefs}
              fitView
              onNodesChange={handleNodesChange}
              onEdgesChange={handleEdgesChange}
              onConnect={handleConnect}
              connectionLineComponent={ConnectionLine}
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

  const Diagram: FC<DiagramProps> = (props) => (
    <ReactFlowProvider>
      <Base {...props} />
    </ReactFlowProvider>
  );

  return Diagram;
};
