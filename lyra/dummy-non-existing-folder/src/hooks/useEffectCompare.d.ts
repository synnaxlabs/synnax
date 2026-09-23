import { type DependencyList, type EffectCallback } from "react";
/**
 * Runs an effect when `areEqual` reports the dependencies changed, instead of on
 * React's reference equality. Use it for deps that are rebuilt every render but rarely
 * change in value, such as an object or array literal.
 */
export declare const useEffectCompare: <D extends DependencyList>(cbk: EffectCallback, areEqual: (prevDeps: D, nextDeps: D) => boolean, deps: D) => void;
//# sourceMappingURL=useEffectCompare.d.ts.map