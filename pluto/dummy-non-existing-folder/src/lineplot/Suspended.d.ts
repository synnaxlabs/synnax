import { type lineplot } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
export declare const useKey: (override?: string | undefined) => string;
export interface SuspendedProps extends PropsWithChildren {
    linePlotKey: lineplot.Key;
}
export declare const Suspended: ({ linePlotKey, children }: SuspendedProps) => ReactElement;
//# sourceMappingURL=Suspended.d.ts.map