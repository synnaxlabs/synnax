import { type schematic } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { type RefCallback } from "react";
export interface UseRenderParams {
    orientation: location.Outer;
    activeState: string;
    externalScale: number;
    spec?: schematic.symbol.Spec;
    onMount?: (svgElement: SVGSVGElement) => void;
    stateOverrides?: schematic.symbol.State[];
}
export declare const useRender: (params: UseRenderParams) => RefCallback<HTMLElement>;
//# sourceMappingURL=render.d.ts.map