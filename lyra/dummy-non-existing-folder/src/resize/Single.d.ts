import { bounds, box } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type BaseProps } from "./Base";
export interface HandlerExtra {
    box: box.Box;
    dragSize: number;
}
/** Props for the {@link Single} component. */
export interface SingleProps extends Omit<BaseProps, "hideHandle" | "size" | "onResize" | "onPointerDown" | "ref"> {
    size?: number;
    sizeBounds?: Partial<bounds.Bounds>;
    onResize?: (size: number, extra: HandlerExtra) => void;
    onResizeEnd?: (size: number, extra: HandlerExtra) => void;
}
export declare const Single: ({ onResize, onResizeEnd, location: propsLoc, sizeBounds, size, className, ...rest }: SingleProps) => ReactElement;
//# sourceMappingURL=Single.d.ts.map