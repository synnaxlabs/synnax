import { type log } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
export declare const useKey: (override?: string | undefined) => string;
export interface SuspendedProps extends PropsWithChildren {
    logKey: log.Key;
}
export declare const Suspended: ({ logKey, children }: SuspendedProps) => ReactElement;
//# sourceMappingURL=Suspended.d.ts.map