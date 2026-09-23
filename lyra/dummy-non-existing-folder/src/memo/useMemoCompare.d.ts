import { type primitive } from "@synnaxlabs/x";
import { type DependencyList } from "react";
export declare const useMemoCompare: <V, D extends DependencyList>(factory: () => V, areEqual: (prevDeps: D, nextDeps: D) => boolean, deps: D) => V;
export declare const compareArrayDeps: <T extends primitive.Value>([a]: readonly [T[]] | [T[]], [b]: readonly [T[]] | [T[]]) => boolean;
export declare const useMemoDeepEqual: <T>(value: T) => T;
/**
 * Returns a referentially stable array as long as its contents are element-wise equal
 * (per `Object.is`, matching React's dep comparison) to the previous render's
 * contents. Faster than {@link useMemoDeepEqual} since it short-circuits on length
 * and avoids structural traversal.
 */
export declare const useMemoArray: <T>(value: readonly T[]) => readonly T[];
//# sourceMappingURL=useMemoCompare.d.ts.map