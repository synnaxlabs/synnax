import { type CrudeTimeSpan } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
export interface ProviderProps extends PropsWithChildren {
    /** How often to check sources for staleness. A bare number is read as milliseconds.
     * Bounds how late a transition can be reported; keep it well under the shortest
     * staleness timeout in use. */
    sweepInterval?: CrudeTimeSpan;
}
export declare const Provider: ({ children, sweepInterval, }: ProviderProps) => ReactElement;
//# sourceMappingURL=Provider.d.ts.map