import { type schematic } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
export declare const useKey: (override?: string | undefined) => string;
export interface SuspendedProps extends PropsWithChildren {
    schematicKey: schematic.Key;
}
export declare const Suspended: ({ schematicKey, children }: SuspendedProps) => ReactElement;
//# sourceMappingURL=Suspended.d.ts.map