import { type Viewport } from "@xyflow/react";
/**
 * Calls `onChange` with every viewport React Flow renders, including the one it mounts
 * at, and with nothing while React Flow is unmounted. Reads the store transform rather
 * than `useOnViewportChange`, whose callbacks React Flow drops when it unmounts.
 */
export declare const useOnViewportChange: (onChange: (viewport: Viewport) => void) => void;
//# sourceMappingURL=useOnViewportChange.d.ts.map