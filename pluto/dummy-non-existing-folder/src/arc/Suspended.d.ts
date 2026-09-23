import { type arc } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
export declare const useKey: (override?: string | undefined) => string;
export interface SuspendedProps extends PropsWithChildren {
    arcKey: arc.Key;
}
export declare const Suspended: ({ arcKey, children }: SuspendedProps) => ReactElement;
//# sourceMappingURL=Suspended.d.ts.map