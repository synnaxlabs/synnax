import { type box, type location, xy } from "@synnaxlabs/x";
import type * as rf from "@xyflow/react";
import type React from "react";
import { z } from "zod/v4";
export declare const viewportZ: z.ZodObject<{
    position: z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>;
    zoom: z.ZodNumber;
}, z.core.$strip>;
export type Viewport = z.infer<typeof viewportZ>;
export declare const handleZ: z.ZodObject<{
    node: z.ZodString;
    param: z.ZodString;
}, z.core.$strip>;
export type Handle = z.infer<typeof handleZ>;
export declare const edgeZ: z.ZodObject<{
    key: z.ZodString;
    source: z.ZodObject<{
        node: z.ZodString;
        param: z.ZodString;
    }, z.core.$strip>;
    target: z.ZodObject<{
        node: z.ZodString;
        param: z.ZodString;
    }, z.core.$strip>;
}, z.core.$strip>;
export type Edge = z.infer<typeof edgeZ>;
export declare const nodeZ: z.ZodObject<{
    key: z.ZodString;
    position: z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>;
    zIndex: z.ZodOptional<z.ZodNumber>;
    type: z.ZodOptional<z.ZodString>;
    draggable: z.ZodOptional<z.ZodBoolean>;
}, z.core.$strip>;
export type Node = z.infer<typeof nodeZ>;
export declare const FIT_VIEW_OPTIONS: FitViewOptions;
export declare const translateNodesForward: (nodes: Node[], selected: Set<string>, dragHandleSelector?: string) => rf.Node[];
export declare const translateEdgesForward: (edges: Edge[], selected: Set<string>) => rf.Edge[];
export declare const translateNodesBackward: (nodes: rf.Node[]) => Node[];
export declare const translateViewportForward: (viewport: Viewport) => rf.Viewport;
export declare const translateViewportBackward: (viewport: rf.Viewport) => Viewport;
export declare const nodeConverter: (nodes: Node[], f: (nodes: rf.Node[]) => rf.Node[]) => Node[];
export type NodeChange = {
    type: "position";
    key: string;
    position: xy.XY;
    dragging: boolean;
} | {
    type: "remove";
    key: string;
} | {
    type: "select";
    key: string;
    selected: boolean;
};
export declare const translateNodeChangeForward: (change: rf.NodeChange) => NodeChange | null;
export type EdgeChange = {
    type: "add";
    edge: Edge;
} | {
    type: "remove";
    key: string;
} | {
    type: "reconnect";
    key: string;
    source: Handle;
    target: Handle;
} | {
    type: "select";
    key: string;
    selected: boolean;
};
export declare const translateEdgeChangeForward: (change: rf.EdgeChange) => EdgeChange | null;
export type FitViewOptions = rf.FitViewOptions;
export interface EdgeEndpoint {
    position: xy.XY;
    orientation: location.Outer;
}
export interface EdgeProps {
    edgeKey: string;
    source: EdgeEndpoint;
    target: EdgeEndpoint;
    sourceNode: string;
    targetNode: string;
    selected: boolean;
}
export interface ConnectionLineProps {
    source: EdgeEndpoint;
    target: EdgeEndpoint;
    sourceBox: box.Box;
    targetBox: box.Box;
    status: "valid" | "invalid" | null;
    style: React.CSSProperties;
}
export declare const createEndpoint: (x: number, y: number, orientation: location.Outer) => EdgeEndpoint;
export declare const createEdgeFromConnection: (connection: rf.Connection) => Edge;
export declare const createReconnect: (key: string, connection: rf.Connection) => EdgeChange;
//# sourceMappingURL=types.d.ts.map