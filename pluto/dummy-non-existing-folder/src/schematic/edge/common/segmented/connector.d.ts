import { type schematic } from "@synnaxlabs/client";
import { box, location, xy } from "@synnaxlabs/x";
import { type diagram } from "../../../../vis/diagram/aether";
export interface CheckIntegrityProps {
    sourcePos: xy.XY;
    targetPos: xy.XY;
    next: Segment[];
    prev: Segment[];
}
export declare const checkIntegrity: ({ sourcePos, targetPos, next, prev, }: CheckIntegrityProps) => boolean;
export interface ChangeOrientationProps {
    orientation: location.Outer;
    segments: Segment[];
}
export declare const changeSourceOrientation: (props: ChangeOrientationProps) => Segment[];
export declare const changeTargetOrientation: (props: ChangeOrientationProps) => Segment[];
export interface PrepareNodeProps {
    sourceStumpTip: xy.XY;
    sourceOrientation: location.Outer;
    sourceBox: box.Box;
    targetStumpTip: xy.XY;
    targetOrientation: location.Outer;
    targetBox: box.Box;
}
export declare const prepareNode: ({ sourceStumpTip: sourcePos, sourceOrientation, sourceBox, targetStumpTip: targetPos, targetOrientation, targetBox, }: PrepareNodeProps) => Segment | undefined;
export type Segment = schematic.Segment;
export declare const travelSegments: (source: xy.XY, ...segments: Segment[]) => xy.XY;
export declare const segmentsToPoints: (source: xy.XY, segments: Segment[], zoom: number, applyTransform: boolean) => xy.XY[];
export interface BuildNew {
    source: diagram.EdgeEndpoint;
    target: diagram.EdgeEndpoint;
    sourceBox: box.Box;
    targetBox: box.Box;
}
export declare const STUMP_LENGTH = 10;
export interface NeedToGoAroundSourceProps {
    sourcePos: xy.XY;
    targetPos: xy.XY;
    sourceOrientation: location.Outer;
}
export declare const needToGoAround: ({ sourcePos, targetPos, sourceOrientation, }: NeedToGoAroundSourceProps) => boolean;
export declare const stump: (orientation: location.Outer) => Segment;
export declare const compressSegments: (segments: Segment[]) => Segment[];
export declare const createConnector: (props: BuildNew) => Segment[];
export interface MoveConnectorProps {
    segments: Segment[];
    index: number;
    magnitude: number;
}
export declare const dragSegment: (props: MoveConnectorProps) => Segment[];
export declare const resolveOrientation: (nodeOrientation: location.Outer, baseLocation: location.Outer) => location.Outer;
export interface BuildNewFromStateProps {
    sourcePos: xy.XY;
    targetPos: xy.XY;
    sourceMeasured?: {
        width: number;
        height: number;
    };
    targetMeasured?: {
        width: number;
        height: number;
    };
    sourceOrientation: location.Outer;
    targetOrientation: location.Outer;
}
export declare const buildNewFromState: ({ sourcePos, targetPos, sourceMeasured, targetMeasured, sourceOrientation, targetOrientation, }: BuildNewFromStateProps) => Segment[];
export interface NodePositionChange {
    key: string;
    newPos: xy.XY;
}
export interface EdgeSegmentUpdate {
    key: string;
    segments: Segment[];
}
export interface UpdateSegmentsForPositionChangesProps {
    nodes: Array<{
        key: string;
        position: xy.XY;
    }>;
    edges: Array<{
        key: string;
        source: {
            node: string;
        };
        target: {
            node: string;
        };
    }>;
    props: Record<string, schematic.ElementConfig | undefined>;
    changes: NodePositionChange[];
}
export declare const updateSegmentsForPositionChanges: ({ nodes, edges, props, changes, }: UpdateSegmentsForPositionChangesProps) => EdgeSegmentUpdate[];
export interface MoveNodeProps {
    delta: xy.XY;
    segments: Segment[];
}
export declare const moveSourceNode: ({ delta, segments }: MoveNodeProps) => Segment[];
export declare const moveTargetNode: ({ delta, segments }: MoveNodeProps) => Segment[];
export interface StitchEdgeProps {
    source: diagram.EdgeEndpoint;
    target: diagram.EdgeEndpoint;
    middleSegments: Segment[];
}
export declare const stitchEdge: ({ source, target, middleSegments, }: StitchEdgeProps) => Segment[];
export interface BuildProps extends BuildNew {
    middleSegments: Segment[];
}
export declare const build: ({ middleSegments, ...endpoints }: BuildProps) => Segment[];
export declare const extractMiddle: (segments: Segment[], sourceOrientation: location.Outer, targetOrientation: location.Outer) => Segment[];
//# sourceMappingURL=connector.d.ts.map