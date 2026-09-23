import { type schematic } from "@synnaxlabs/client";
export declare const VISUAL_ELEMENTS: string[];
export declare const extract: (svgElement: SVGElement) => schematic.symbol.Region[];
export interface SymbolColors {
    fill: string | null;
    stroke: string | null;
}
export type ColorResolver = (el: SVGElement) => SymbolColors;
export declare const normalizeElement: (svg: SVGSVGElement, resolveColors: ColorResolver) => void;
export declare const normalizeSVG: (svgString: string) => string;
//# sourceMappingURL=region.d.ts.map