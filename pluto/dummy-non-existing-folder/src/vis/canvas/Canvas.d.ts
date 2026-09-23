import "./Canvas.css";
import { type UseResizeHandler, type UseResizeOpts } from "@synnaxlabs/lyra/hooks";
import { type CrudeTimeSpan } from "@synnaxlabs/x";
import { type CanvasHTMLAttributes, type DetailedHTMLProps, type ReactElement, type RefCallback } from "react";
type HTMLDivProps = DetailedHTMLProps<CanvasHTMLAttributes<HTMLDivElement>, HTMLDivElement>;
export interface CanvasProps extends Omit<HTMLDivProps, "ref"> {
    resizeDebounce?: CrudeTimeSpan;
}
export declare const Canvas: ({ children, resizeDebounce: debounce, className, ...rest }: CanvasProps) => ReactElement;
export declare const useRegion: (handler: UseResizeHandler, opts?: UseResizeOpts) => RefCallback<HTMLDivElement>;
export {};
//# sourceMappingURL=Canvas.d.ts.map