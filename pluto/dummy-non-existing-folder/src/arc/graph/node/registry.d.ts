import { Icon } from "@synnaxlabs/lyra/icon";
import z from "zod/v4";
import { type Spec } from "./types/spec";
export declare const REGISTRY: {
    readonly constant: Spec<"constant", import("./constant/config").Config>;
    readonly add: Spec<"add", {
        type: "add";
    }>;
    readonly subtract: Spec<"subtract", {
        type: "subtract";
    }>;
    readonly multiply: Spec<"multiply", {
        type: "multiply";
    }>;
    readonly divide: Spec<"divide", {
        type: "divide";
    }>;
    readonly gt: Spec<"gt", {
        type: "gt";
    }>;
    readonly lt: Spec<"lt", {
        type: "lt";
    }>;
    readonly eq: Spec<"eq", {
        type: "eq";
    }>;
    readonly ne: Spec<"ne", {
        type: "ne";
    }>;
    readonly ge: Spec<"ge", {
        type: "ge";
    }>;
    readonly le: Spec<"le", {
        type: "le";
    }>;
    readonly and: Spec<"and", {
        type: "and";
    }>;
    readonly or: Spec<"or", {
        type: "or";
    }>;
    readonly not: Spec<"not", {
        type: "not";
    }>;
    readonly select: Spec<"select", import("./select/config").Config>;
    readonly write: Spec<"write", import("./sink/config").Config>;
    readonly on: Spec<"on", import("./source/config").Config>;
    readonly stable_for: Spec<"stable_for", import("./stable/config").Config>;
    readonly "status.set": Spec<"status.set", import("./status/config").Config>;
};
export type Type = keyof typeof REGISTRY;
export declare const configZ: z.ZodDiscriminatedUnion<[z.ZodObject<{
    type: z.ZodLiteral<"on">;
    channel: z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"write">;
    channel: z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>;
    value: z.ZodNumber;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"constant">;
    value: z.ZodNumber;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"select">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"status.set">;
    key_or_name: z.ZodString;
    variant: z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>;
    message: z.ZodString;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"add">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"subtract">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"multiply">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"divide">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"gt">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"lt">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"eq">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"ne">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"ge">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"le">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"and">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"or">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"not">;
}, z.core.$strip>, z.ZodObject<{
    type: z.ZodLiteral<"stable_for">;
    duration: z.ZodNumber;
}, z.core.$strip>], "type">;
export type Config = z.infer<typeof configZ>;
export type ConfigOf<T extends Type> = Extract<Config, {
    type: T;
}>;
export declare const resolveSpec: (type: string) => Spec<Type, Config>;
export interface Group {
    key: string;
    name: string;
    Icon: Icon.FC;
    symbols: Type[];
}
export declare const GROUPS: Group[];
//# sourceMappingURL=registry.d.ts.map