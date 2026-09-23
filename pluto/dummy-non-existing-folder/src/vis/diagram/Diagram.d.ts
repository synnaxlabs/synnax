import "./Diagram.css";
import "@xyflow/react/dist/base.css";
import { type Component } from "@synnaxlabs/lyra/component";
import { xy } from "@synnaxlabs/x";
import { type ReactFlowProps } from "@xyflow/react";
import { type ClipboardEvent as ReactClipboardEvent, type ComponentPropsWithRef, type FC, type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";
import { Aether } from "../../aether";
import { Viewport as BaseViewport } from "../../viewport";
import { diagram } from "./aether";
import { type Edge, type EdgeChange, type Node, type NodeChange, type Viewport } from "./aether/types";
export interface NodeProps {
    nodeKey: string;
    position: xy.XY;
    selected: boolean;
    draggable: boolean;
}
export interface RendererConfig {
    /** Renders each node by key. */
    node: Component.RenderProp<NodeProps, ReactElement>;
    /** Renders each edge; falls back to React Flow's default edge when omitted. */
    edge?: Component.RenderProp<diagram.EdgeProps, ReactElement>;
    /** Renders the line shown while dragging a new connection. */
    connectionLine?: Component.RenderProp<diagram.ConnectionLineProps, ReactElement>;
    /** Wraps the diagram inside the React Flow store context, above all renderers. */
    Provider?: FC<PropsWithChildren>;
}
export type ClipboardHandler = (this: void, e: ReactClipboardEvent<HTMLDivElement>, cursor: xy.XY) => void;
export interface DiagramProps extends Omit<ComponentPropsWithRef<"div">, "onError" | "onCopy" | "onCut" | "onPaste">, Pick<z.infer<typeof diagram.Diagram.stateZ>, "visible" | "autoRenderInterval">, Aether.ComponentProps, Pick<ReactFlowProps, "minZoom" | "maxZoom" | "fitViewOptions" | "snapGrid" | "snapToGrid" | "onNodeClick" | "onNodeDoubleClick" | "edgesReconnectable"> {
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
    /**
     * Called when a copy event fires on the diagram. The second argument is the
     * cursor position in diagram space at the moment of the copy, derived from
     * the most recent mousemove over the diagram.
     */
    onCopy?: ClipboardHandler;
    /**
     * Called when a cut event fires on the diagram. The second argument is the
     * cursor position in diagram space at the moment of the cut, derived from
     * the most recent mousemove over the diagram. Ignored when not editable.
     */
    onCut?: ClipboardHandler;
    /**
     * Called when a paste event fires on the diagram. The second argument is the
     * cursor position in diagram space at the moment of the paste, derived from
     * the most recent mousemove over the diagram.
     */
    onPaste?: ClipboardHandler;
}
export declare const create: ({ node: nodeRenderer, edge: edgeRenderer, connectionLine: connectionLineRenderer, Provider, }: RendererConfig) => FC<DiagramProps>;
//# sourceMappingURL=Diagram.d.ts.map