import { type CrudeTimeSpan, debounce } from "@synnaxlabs/x";
import { type DependencyList } from "react";
/**
 * @returns a stable callback that defers `func` until `waitFor` passes with no further
 * calls. The identity changes only when `waitFor` or `deps` change, so a pending call
 * survives a re-render.
 */
export declare const useDebouncedCallback: <Args extends unknown[]>(func: (...args: Args) => void, waitFor: CrudeTimeSpan, deps: DependencyList) => debounce.DebouncedFn<Args>;
//# sourceMappingURL=useDebouncedCallback.d.ts.map