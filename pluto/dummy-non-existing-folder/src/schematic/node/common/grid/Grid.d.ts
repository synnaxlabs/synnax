import "./grid.css";
import { type dimensions, location } from "@synnaxlabs/x";
import { type ControlLinePosition, type ControlPosition } from "@xyflow/react";
import { type CSSProperties, type DragEvent, type FC, type PropsWithChildren, type ReactElement } from "react";
type DraggableElement = ReactElement<{
    style?: CSSProperties;
    draggable?: boolean;
    onDragStart?: (e: DragEvent<HTMLElement>) => void;
    onDragEnd?: (e: DragEvent<HTMLElement>) => void;
}>;
export interface ItemProps {
    itemKey: string;
    location: location.Location;
    onLocationChange?: (loc: location.Location) => void;
    children: DraggableElement;
}
export declare const Item: FC<ItemProps>;
export declare const createItem: <P extends {}>(component: FC<P>) => FC<P>;
export interface GridProps extends PropsWithChildren<{}> {
    editable: boolean;
    nodeKey: string;
    orientation?: location.Outer;
    onRotate?: (params: {
        orientation: location.Outer;
    }) => void;
    allowCenter?: boolean;
    allowRotate?: boolean;
    onResize?: (dimensions: dimensions.Dimensions) => void;
    keepAspectRatio?: boolean;
    resizeHandles?: (ControlLinePosition | ControlPosition)[];
    onResizeStart?: (dimensions: dimensions.Dimensions) => void;
}
export declare const useScaleResize: (config: {
    scale?: number;
}, onConfigChange: (config: {
    scale?: number;
}) => void) => Pick<GridProps, "keepAspectRatio" | "onResizeStart" | "onResize">;
export declare const DRAG_HANDLE_CLASS: string;
export declare const Grid: FC<GridProps>;
export {};
//# sourceMappingURL=Grid.d.ts.map