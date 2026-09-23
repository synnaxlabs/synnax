import { connection, Synnax, type SynnaxParams } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";
import z from "zod";
import { synnax } from "./aether";
export interface ContextValue extends synnax.ContextValue {
    status: connection.Status;
}
export declare const use: () => Synnax | null;
export declare const useConnectionStatus: () => connection.Status;
export interface ProviderProps extends PropsWithChildren {
    connParams?: SynnaxParams;
}
export declare const SERVER_VERSION_MISMATCH = "serverVersionMismatch";
export declare const CLOCK_SKEW_EXCEEDED = "clockSkewExceeded";
export declare const statusDetailsSchema: z.ZodObject<{
    type: z.ZodString;
    oldServer: z.ZodBoolean;
    nodeVersion: z.ZodOptional<z.ZodString>;
    clientVersion: z.ZodString;
}, z.core.$strip>;
export declare const clockSkewDetailsSchema: z.ZodObject<{
    type: z.ZodString;
    clockSkew: z.ZodNumber;
}, z.core.$strip>;
export interface StatusDetails extends z.infer<typeof statusDetailsSchema> {
}
interface TestProviderProps extends PropsWithChildren {
    client: Synnax | null;
    status?: connection.Status;
}
export declare const TestProvider: ({ children, client, status, }: TestProviderProps) => ReactElement;
export declare const Provider: ({ children, connParams: connParamsProp, }: ProviderProps) => ReactElement;
export {};
//# sourceMappingURL=Provider.d.ts.map