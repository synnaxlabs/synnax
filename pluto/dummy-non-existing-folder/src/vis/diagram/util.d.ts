import { box, dimensions, location, xy } from "@synnaxlabs/x";
import { type InternalNode, type NodeChange as RFNodeChange, type ReactFlowInstance } from "@xyflow/react";
import { type diagram } from "./aether";
import { type Diagram } from ".";
export declare const selectNode: (key: string) => HTMLDivElement;
export declare const selectNodesScreenBounds: (root: HTMLElement) => box.Box | null;
export declare const internalNodeBox: (node: InternalNode | null) => box.Box;
interface HandleGeometry {
    id?: string | null;
    position: string;
    x: number;
    y: number;
    width: number;
    height: number;
}
export interface NodeGeometry {
    positionAbsolute: xy.XY;
    handleBounds?: {
        source?: HandleGeometry[] | null;
        target?: HandleGeometry[] | null;
    } | null;
}
export declare const resolveEndpoint: ({ positionAbsolute, handleBounds }: NodeGeometry, handleKey: string) => diagram.EdgeEndpoint | null;
export declare const selectNodeBox: (flow: ReactFlowInstance, key: string) => box.Box;
export declare const selectNodeLayout: (key: string, flow: ReactFlowInstance) => NodeLayout;
export declare class HandleLayout {
    node_: NodeLayout | null;
    position: xy.XY;
    orientation: location.Outer;
    constructor(position: xy.XY, orientation: location.Outer);
    set node(node: NodeLayout);
    get node(): NodeLayout;
    get absolutePosition(): xy.XY;
}
export declare class NodeLayout {
    key: string;
    box: box.Box;
    handles: HandleLayout[];
    constructor(key: string, box: box.Box, handles: HandleLayout[]);
    static fromFlow(key: string, flow: ReactFlowInstance): NodeLayout;
}
export declare const calculateCursorPosition: (region: box.Box, cursor: xy.Crude, viewport: Diagram.Viewport) => xy.XY;
export interface PartitionedNodeChanges {
    passthrough: RFNodeChange[];
    sizes: [string, dimensions.Dimensions][];
    removed: string[];
}
export declare const partitionNodeChanges: (changes: RFNodeChange[]) => PartitionedNodeChanges;
export {};
//# sourceMappingURL=util.d.ts.map