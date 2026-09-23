import { type table } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
export declare const useKey: (override?: string | undefined) => string;
export interface SuspendedProps extends PropsWithChildren {
    tableKey: table.Key;
}
export declare const Suspended: ({ tableKey, children }: SuspendedProps) => ReactElement;
//# sourceMappingURL=Suspended.d.ts.map