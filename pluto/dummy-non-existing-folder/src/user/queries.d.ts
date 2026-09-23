import { user } from "@synnaxlabs/client";
import { z } from "zod";
import { Flux } from "../flux";
export type UseDeleteParams = user.Key | user.Key[];
export declare const useDelete: Flux.UseUpdate<UseDeleteParams, UseDeleteParams, z.ZodNever>;
export type RetrieveQuery = {
    key: user.Key;
};
export interface ChangeUsernameParams extends Pick<user.User, "key" | "username"> {
}
export declare const useRename: Flux.UseUpdate<ChangeUsernameParams, ChangeUsernameParams, z.ZodNever>;
export type UseRetrieveGroupParams = Record<string, never>;
export declare const useGroupID: Flux.Use<UseRetrieveGroupParams, {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | undefined>;
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    username: z.ZodString;
    password: z.ZodString;
    firstName: z.ZodString;
    lastName: z.ZodString;
    role: z.ZodUUID;
}, z.core.$strip>;
export type FormQuery = {
    key: user.Key;
};
export declare const useForm: Flux.UseForm<FormQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    username: z.ZodString;
    password: z.ZodString;
    firstName: z.ZodString;
    lastName: z.ZodString;
    role: z.ZodUUID;
}, z.core.$strip>>;
export declare const use: Flux.Use<Partial<RetrieveQuery>, user.User>, useResult: Flux.UseResult<Partial<RetrieveQuery>, user.User>, createResultSelector: Flux.CreateResultSelector<Partial<RetrieveQuery>, user.User>;
export declare const useResultKey: Flux.UseResult<Partial<RetrieveQuery>, string>;
export declare const useResultUsername: Flux.UseResult<Partial<RetrieveQuery>, string>;
export declare const useResultFirstName: Flux.UseResult<Partial<RetrieveQuery>, string>;
export declare const useResultLastName: Flux.UseResult<Partial<RetrieveQuery>, string>;
//# sourceMappingURL=queries.d.ts.map