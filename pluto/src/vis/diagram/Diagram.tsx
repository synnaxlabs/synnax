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
  memo,
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
import { useCombinedRefs, useDebouncedCallback, useSyncedRef } from "@/hooks";
import { useMemoCompare } from "@/memo";
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
  const NodeWrapper = memo(
    ({
      id,
      positionAbsoluteX: x,
      positionAbsoluteY: y,
      selected = false,
      draggable = true,
    }: RFNodeProps): ReactElement => {
      const position = useMemo(() => ({ x, y }), [x, y]);
      return nodeRenderer({ nodeKey: id, position, selected, draggable });
    },
  );
  NodeWrapper.displayName = "NodeWrapper";

  const nodeTypes = { custom: NodeWrapper };

  const EdgeWrapper = memo(
    ({
      id,
      sourceX,
      sourceY,
      sourcePosition,
      targetX,
      targetY,
      targetPosition,
      source,
      target,
      selected = false,
    }: RFEdgeProps): ReactElement => {
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
    },
  );
  EdgeWrapper.displayName = "EdgeWrapper";

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
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: diagram.Diagram.TYPE,
      schema: diagram.diagramStateZ,
      initialState: {
        position: viewport.position,
        region: box.ZERO,
        zoom: viewport.zoom,
        visible,
        autoRenderInterval,
      },
    });
    useEffect(
      () => setState((prev) => ({ ...prev, visible, autoRenderInterval })),
      [visible, autoRenderInterval],
    );

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

    const triggers = useMemoCompare(
      () => pTriggers ?? BaseViewport.DEFAULT_TRIGGERS.zoom,
      Triggers.compareModeConfigs,
      [pTriggers],
    );

    const zoomRef = useRef<number>(viewport.zoom);
    const syncZoomCSSVar = useCallback((zoom: number): void => {
      if (zoomRef.current === zoom) return;
      zoomRef.current = zoom;
      triggerRef.current?.style.setProperty(CSS.var("diagram-zoom"), `${zoom}`);
    }, []);
    syncZoomCSSVar(viewport.zoom);

    const viewportRef = useRef<RFViewport | null>(null);
    const handleViewportChange = useCallback(
      (vp: RFViewport): void => {
        const prev = viewportRef.current;
        if (prev != null && prev.x === vp.x && prev.y === vp.y && prev.zoom === vp.zoom)
          return;
        viewportRef.current = vp;
        if (isNaN(vp.x) || isNaN(vp.y) || isNaN(vp.zoom)) return;
        syncZoomCSSVar(vp.zoom);
        setState((prev) => ({ ...prev, position: vp, zoom: vp.zoom }));
        onViewportChange(translateViewportBackward(vp));
      },
      [setState, onViewportChange, syncZoomCSSVar],
    );

    useRFOnViewportChange({
      onStart: handleViewportChange,
      onChange: handleViewportChange,
      onEnd: handleViewportChange,
    });

    const selectedSet = useMemo(() => new Set(selected), [selected]);
    const selectedRef = useSyncedRef(selectedSet);

    const rfEdges = useMemo(
      () => translateEdgesForward(edges, selectedSet),
      [edges, selectedSet],
    );
    const rfNodes = useMemo(
      () => translateNodesForward(nodes, selectedSet, dragHandleSelector),
      [nodes, selectedSet, dragHandleSelector],
    );

    const processChanges = useCallback(
      <
        RFChange extends RFEdgeChange | RFNodeChange,
        Change extends EdgeChange | NodeChange,
      >(
        rfChanges: RFChange[],
        translate: (c: RFChange) => Change | null,
        onMutations: (changes: Change[]) => void,
      ): void => {
        const mutations: Change[] = [];
        let nextSelected: Set<string> | undefined;
        for (const change of rfChanges) {
          const translated = translate(change);
          if (translated == null) continue;
          if (translated.type === "select") {
            if (nextSelected === undefined) nextSelected = new Set(selectedRef.current);
            if (translated.selected) nextSelected.add(translated.key);
            else nextSelected.delete(translated.key);
            continue;
          }
          if (translated.type === "remove") {
            if (nextSelected === undefined) nextSelected = new Set(selectedRef.current);
            nextSelected.delete(translated.key);
          }
          mutations.push(translated);
        }
        if (nextSelected !== undefined) onSelectionChange?.([...nextSelected]);
        if (mutations.length > 0) onMutations(mutations);
      },
      [onSelectionChange],
    );

    const handleNodesChange = useCallback(
      (changes: RFNodeChange[]) =>
        processChanges(changes, translateNodeChangeForward, onNodesChange),
      [processChanges, onNodesChange],
    );

    const handleEdgesChange = useCallback(
      (changes: RFEdgeChange[]) =>
        processChanges(changes, translateEdgeChangeForward, onEdgesChange),
      [processChanges, onEdgesChange],
    );

    const handleConnect = useCallback(
      (conn: RFConnection) =>
        onEdgesChange([{ type: "add", edge: diagram.createEdgeFromConnection(conn) }]),
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

    const triggerProps = useMemo<Partial<ReactFlowProps>>(() => {
      const selectTriggers = Triggers.purgeMouse(triggers.select)[0] ?? null;
      const panTriggers = Triggers.purgeMouse(triggers.pan)[0] ?? null;
      const zoomTriggers = Triggers.purgeMouse(triggers.zoom)[0] ?? null;
      return {
        selectionOnDrag: selectTriggers == null,
        panOnDrag: panTriggers == null,
        selectionKeyCode: selectTriggers,
        panActivationKeyCode: panTriggers,
        zoomActivationKeyCode: zoomTriggers,
      };
    }, [triggers]);

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

    const defautlViewport = useMemo(
      () => translateViewportForward(viewport),
      [viewport],
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
              defaultViewport={defautlViewport}
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

  return memo(Diagram);
};
