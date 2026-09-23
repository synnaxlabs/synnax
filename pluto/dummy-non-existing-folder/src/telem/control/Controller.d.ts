import { type channel } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";
import { control } from "./aether";
export interface ControllerProps extends Omit<z.input<typeof control.controllerStateZ>, "needsControlOf">, PropsWithChildren {
    onStatusChange?: (status: control.Status) => void;
    name: string;
}
export interface ContextValue {
    key: string;
    needsControlOf: channel.Key[];
    acquire: () => void;
    release: () => void;
    status: control.Status;
}
declare const useContext: () => ContextValue;
export { useContext };
export declare const Controller: ({ children, onStatusChange, disabled, ...props }: ControllerProps) => ReactElement;
//# sourceMappingURL=Controller.d.ts.map