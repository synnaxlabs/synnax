import { type direction, location } from "@synnaxlabs/x";
import { type NodeLayout } from "./util";
/** Aligns nodes to a specific edge (left/right/top/bottom). */
export declare const alignNodesToLocation: (layouts: NodeLayout[], loc: location.Outer) => NodeLayout[];
/** Aligns nodes by their handles along a direction (x or y). */
export declare const alignNodesAlongDirection: (layouts: NodeLayout[], dir?: direction.Direction) => NodeLayout[];
export declare const distributeNodes: (layouts: NodeLayout[], dir: direction.Direction) => NodeLayout[];
export declare const rotateNodesAroundCenter: (layouts: NodeLayout[], dir: direction.Angular) => NodeLayout[];
//# sourceMappingURL=align.d.ts.map