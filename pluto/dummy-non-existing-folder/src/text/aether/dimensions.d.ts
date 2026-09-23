import { type dimensions as base } from "@synnaxlabs/x";
/**
 * Measures the rendered dimensions of a text string in the given CSS font.
 * Falls back to zero dimensions when no DOM is available (e.g., node).
 */
export declare const dimensions: (text: string, font: string, context?: OffscreenCanvasRenderingContext2D | CanvasRenderingContext2D) => base.Dimensions;
/** Derives integer pixel dimensions from a canvas TextMetrics result. */
export declare const dimensionsFromMetrics: (metrics: TextMetrics) => base.Dimensions;
//# sourceMappingURL=dimensions.d.ts.map