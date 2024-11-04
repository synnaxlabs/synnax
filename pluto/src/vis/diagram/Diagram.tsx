// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/diagram/Diagram.css";
import "reactflow/dist/style.css";

import { Icon } from "@synnaxlabs/media";
import { box, location, xy } from "@synnaxlabs/x";
import {
  type ComponentPropsWithoutRef,
  createContext,
  memo,
  type ReactElement,
  useCallback,
  useContext as reactUseContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import ReactFlow, {
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
  type NodeChange as RFNodeChange,
  type NodeProps as RFNodeProps,
  type ProOptions,
  type ReactFlowProps,
  ReactFlowProvider,
  reconnectEdge,
  SelectionMode,
  useOnViewportChange as useRFOnViewportChange,
  useReactFlow,
  type Viewport as RFViewport,
} from "reactflow";
import { z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useCombinedRefs, useDebouncedCallback, useSyncedRef } from "@/hooks";
import { useMemoCompare, useMemoDeepEqualProps } from "@/memo";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";
import { type RenderProp } from "@/util/renderProp";
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
import { Edge as EdgeComponent } from "@/vis/diagram/edge";
import { type connector } from "@/vis/diagram/edge/connector";
import { CustomConnectionLine } from "@/vis/diagram/edge/Edge";

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
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
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
  maxZoom: 1,
  minZoom: 0.5,
};

const PRO_OPTIONS: ProOptions = {
  hideAttribution: true,
};

export interface DiagramProps
  extends UseReturn,
    Omit<ComponentPropsWithoutRef<"div">, "onError">,
    Pick<z.infer<typeof diagram.Diagram.stateZ>, "visible"> {
  triggers?: CoreViewport.UseTriggers;
}

interface ContextValue {
  editable: boolean;
  visible: boolean;
  onEditableChange: (v: boolean) => void;
  registerNodeRenderer: (renderer: RenderProp<SymbolProps>) => void;
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
}

const Context = createContext<ContextValue>({
  editable: true,
  visible: true,
  onEditableChange: () => {},
  registerNodeRenderer: () => {},
  fitViewOnResize: false,
  setFitViewOnResize: () => {},
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
    fitViewOnResize,
    setFitViewOnResize,
    visible,
    ...props
  }): ReactElement => {
    const memoProps = useMemoDeepEqualProps({ visible });
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
    useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);

    const defaultEdgeColor = Theming.use().colors.gray.l9.hex;

    const triggers = useMemoCompare(
      () => pTriggers ?? CoreViewport.DEFAULT_TRIGGERS.zoom,
      Triggers.compareModeConfigs,
      [pTriggers],
    );

    const { fitView } = useReactFlow();
    const debouncedFitView = useDebouncedCallback((args) => fitView(args), 50, [
      fitView,
    ]);
    const resizeRef = Canvas.useRegion(
      useCallback(
        (b) => {
          if (fitViewOnResize) debouncedFitView({ maxZoom: 1 });
          setState((prev) => ({ ...prev, region: b }));
        },
        [setState, debouncedFitView, fitViewOnResize],
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
        custom: ({ id, xPos: x, yPos: y, selected }: RFNodeProps) => {
          return renderer({ symbolKey: id, position: { x, y }, selected });
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
            (e) => reconnectEdge(oldEdge, newConnection, e),
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

    const handleEdgeSegmentsChangeRef = useSyncedRef(handleEdgeSegmentsChange);

    const edgeTypes = useMemo(
      () => ({
        default: (props: RFEdgeProps) => (
          <EdgeComponent
            key={props.id}
            {...props}
            segments={props.data.segments}
            color={props.data.color}
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

    const ctxValue = useMemo(
      () => ({
        visible,
        editable,
        onEditableChange,
        registerNodeRenderer,
        fitViewOnResize,
        setFitViewOnResize,
      }),
      [editable, visible, onEditableChange, registerNodeRenderer, fitViewOnResize],
    );

    return (
      <Context.Provider value={ctxValue}>
        <Aether.Composite path={path}>
          {visible && (
            <ReactFlow
              {...triggerProps}
              className={CSS(CSS.B("diagram"), CSS.editable(editable))}
              nodes={nodes_}
              edges={edges_}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
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
          )}
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
  <Align.Pack
    direction="y"
    borderShade={4}
    className={CSS.BE("diagram", "controls")}
    {...props}
  >
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
  const { fitViewOnResize, setFitViewOnResize } = useContext();
  return (
    <Button.ToggleIcon
      onClick={(e) => {
        fitView(FIT_VIEW_OPTIONS);
        onClick?.(e);
      }}
      // @ts-expect-error - toggle icon issues
      value={fitViewOnResize}
      // @ts-expect-error - toggle icon issues
      onChange={(v) => setFitViewOnResize(v)}
      rightClickToggle
      tooltip={<Text.Text level="small">Fit view to contents</Text.Text>}
      tooltipLocation={location.RIGHT_CENTER}
      variant="outlined"
      {...props}
    >
      <Icon.Expand />
    </Button.ToggleIcon>
  );
};

export const Diagram = (props: DiagramProps): ReactElement => (
  <ReactFlowProvider>
    <Core {...props} />
  </ReactFlowProvider>
);
