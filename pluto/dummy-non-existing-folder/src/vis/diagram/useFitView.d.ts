import { type diagram } from "./aether";
/**
 * Returns a replacement for React Flow's fitView that fits the viewport around
 * everything the nodes render, not just their measured boxes, so content placed
 * outside a node (a label) stays in view. Options follow React Flow's fitView.
 */
export declare const useFitView: () => ((options?: diagram.FitViewOptions) => void);
/**
 * Fits the view once per mount while enabled, deferred until React Flow has measured
 * the nodes, so a diagram that mounts empty still fits when its first nodes arrive.
 * Disabling resets the fit, so the next enable fits again.
 */
export declare const useInitialFitView: (enabled: boolean, options?: diagram.FitViewOptions) => void;
//# sourceMappingURL=useFitView.d.ts.map