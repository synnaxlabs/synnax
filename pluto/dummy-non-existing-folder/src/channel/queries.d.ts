import { channel, DataType, type group, type ranger } from "@synnaxlabs/client";
import { control, type optional, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";
import { type RetrieveMultipleQuery, type RetrieveQuery } from "./aether/queries";
import { Flux } from "../flux";
export declare const formSchema: z.ZodObject<{
    key: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    leaseholder: z.ZodDefault<z.ZodInt>;
    isIndex: z.ZodDefault<z.ZodBoolean>;
    index: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    alias: z.ZodOptional<z.ZodString>;
    virtual: z.ZodDefault<z.ZodBoolean>;
    internal: z.ZodDefault<z.ZodBoolean>;
    expression: z.ZodNonOptional<z.ZodDefault<z.ZodString>>;
    operations: z.ZodDefault<z.ZodArray<z.ZodObject<{
        type: z.ZodEnum<{
            avg: "avg";
            derivative: "derivative";
            max: "max";
            min: "min";
            none: "none";
        }>;
        resetChannel: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
        duration: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    }, z.core.$strip>>>;
    concurrency: z.ZodDefault<z.ZodEnum<typeof control.Concurrency>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodNever, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    name: z.ZodString;
    dataType: z.ZodPipe<z.ZodUnion<readonly [z.ZodCustom<DataType, DataType>, z.ZodPipe<z.ZodString, z.ZodTransform<DataType, string>>, z.ZodPipe<z.ZodObject<{
        value: z.ZodString;
    }, z.core.$strip>, z.ZodTransform<DataType, {
        value: string;
    }>>]>, z.ZodTransform<string, DataType>>;
    requires: z.ZodOptional<z.ZodArray<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>>;
}, z.core.$strip>;
export declare const calculatedFormSchema: z.ZodObject<{
    key: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    leaseholder: z.ZodDefault<z.ZodInt>;
    isIndex: z.ZodDefault<z.ZodBoolean>;
    index: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    alias: z.ZodOptional<z.ZodString>;
    virtual: z.ZodDefault<z.ZodBoolean>;
    internal: z.ZodDefault<z.ZodBoolean>;
    operations: z.ZodDefault<z.ZodArray<z.ZodObject<{
        type: z.ZodEnum<{
            avg: "avg";
            derivative: "derivative";
            max: "max";
            min: "min";
            none: "none";
        }>;
        resetChannel: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
        duration: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    }, z.core.$strip>>>;
    concurrency: z.ZodDefault<z.ZodEnum<typeof control.Concurrency>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodNever, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    name: z.ZodString;
    dataType: z.ZodPipe<z.ZodUnion<readonly [z.ZodCustom<DataType, DataType>, z.ZodPipe<z.ZodString, z.ZodTransform<DataType, string>>, z.ZodPipe<z.ZodObject<{
        value: z.ZodString;
    }, z.core.$strip>, z.ZodTransform<DataType, {
        value: string;
    }>>]>, z.ZodTransform<string, DataType>>;
    requires: z.ZodOptional<z.ZodArray<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>>;
    expression: z.ZodString;
}, z.core.$strip>;
export { type RetrieveMultipleQuery, type RetrieveQuery, } from "./aether/queries";
export declare const ZERO_FORM_VALUES: z.infer<typeof formSchema | typeof calculatedFormSchema>;
export declare const use: Flux.Use<RetrieveQuery, channel.Channel>, useResult: Flux.UseResult<RetrieveQuery, channel.Channel>, useEnsure: Flux.UseEnsure<RetrieveQuery>, createSelector: Flux.CreateSelector<RetrieveQuery, channel.Channel>, createResultSelector: Flux.CreateResultSelector<RetrieveQuery, channel.Channel>;
/** The channel's range-scoped alias when one is set, its name otherwise. */
export declare const useAlias: Flux.UseSelect<RetrieveQuery, string>;
/** {@link useAlias} with the result contract: fetch on cold, never throw. */
export declare const useResultAlias: Flux.UseResult<RetrieveQuery, string>;
export declare const useResultName: Flux.UseResult<RetrieveQuery, string>;
/** Compared by variant and message: a heartbeat that changes neither is silenced. */
export declare const useResultStatus: Flux.UseResult<RetrieveQuery, channel.Status | undefined>;
export declare const useResultDataType: Flux.UseResult<RetrieveQuery, DataType>;
/** The stored range-scoped alias (undefined when unset) alongside the name. */
export declare const useResultAliasAndName: Flux.UseResult<RetrieveQuery, {
    alias: string | undefined;
    name: string;
}>;
export declare const useMultiple: Flux.Use<RetrieveMultipleQuery, channel.Channel[]>, useResultMultiple: Flux.UseResult<RetrieveMultipleQuery, channel.Channel[]>;
export type FormQuery = RetrieveQuery;
export declare const useForm: Flux.UseForm<RetrieveQuery, z.ZodObject<{
    key: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    leaseholder: z.ZodDefault<z.ZodInt>;
    isIndex: z.ZodDefault<z.ZodBoolean>;
    index: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    alias: z.ZodOptional<z.ZodString>;
    virtual: z.ZodDefault<z.ZodBoolean>;
    internal: z.ZodDefault<z.ZodBoolean>;
    expression: z.ZodNonOptional<z.ZodDefault<z.ZodString>>;
    operations: z.ZodDefault<z.ZodArray<z.ZodObject<{
        type: z.ZodEnum<{
            avg: "avg";
            derivative: "derivative";
            max: "max";
            min: "min";
            none: "none";
        }>;
        resetChannel: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
        duration: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    }, z.core.$strip>>>;
    concurrency: z.ZodDefault<z.ZodEnum<typeof control.Concurrency>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodNever, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    name: z.ZodString;
    dataType: z.ZodPipe<z.ZodUnion<readonly [z.ZodCustom<DataType, DataType>, z.ZodPipe<z.ZodString, z.ZodTransform<DataType, string>>, z.ZodPipe<z.ZodObject<{
        value: z.ZodString;
    }, z.core.$strip>, z.ZodTransform<DataType, {
        value: string;
    }>>]>, z.ZodTransform<string, DataType>>;
    requires: z.ZodOptional<z.ZodArray<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>>;
}, z.core.$strip>>;
export declare const useCalculatedForm: Flux.UseForm<RetrieveQuery, z.ZodObject<{
    key: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    leaseholder: z.ZodDefault<z.ZodInt>;
    isIndex: z.ZodDefault<z.ZodBoolean>;
    index: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
    alias: z.ZodOptional<z.ZodString>;
    virtual: z.ZodDefault<z.ZodBoolean>;
    internal: z.ZodDefault<z.ZodBoolean>;
    operations: z.ZodDefault<z.ZodArray<z.ZodObject<{
        type: z.ZodEnum<{
            avg: "avg";
            derivative: "derivative";
            max: "max";
            min: "min";
            none: "none";
        }>;
        resetChannel: z.ZodDefault<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>;
        duration: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
    }, z.core.$strip>>>;
    concurrency: z.ZodDefault<z.ZodEnum<typeof control.Concurrency>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodNever, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    name: z.ZodString;
    dataType: z.ZodPipe<z.ZodUnion<readonly [z.ZodCustom<DataType, DataType>, z.ZodPipe<z.ZodString, z.ZodTransform<DataType, string>>, z.ZodPipe<z.ZodObject<{
        value: z.ZodString;
    }, z.core.$strip>, z.ZodTransform<DataType, {
        value: string;
    }>>]>, z.ZodTransform<string, DataType>>;
    requires: z.ZodOptional<z.ZodArray<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>>;
    expression: z.ZodString;
}, z.core.$strip>>;
export type ListQuery = channel.RetrieveOptions & {
    searchTerm?: string;
    rangeKey?: string;
    internal?: boolean;
    offset?: number;
    limit?: number;
};
export declare const useList: Flux.UseList<ListQuery, number, channel.Channel>;
export interface RenameParams extends Pick<channel.Payload, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export interface UpdateAliasParams extends optional.Optional<ranger.alias.Alias, "range" | "channel"> {
    alias: string;
}
export declare const useUpdateAlias: Flux.UseUpdate<UpdateAliasParams, UpdateAliasParams, z.ZodNever>;
export type DeleteParams = channel.Key | channel.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export interface DeleteAliasParams {
    range?: ranger.Key;
    channels?: channel.Key | channel.Key[];
}
export declare const useDeleteAlias: Flux.UseUpdate<DeleteAliasParams, DeleteAliasParams, z.ZodNever>;
type RetrieveGroupQuery = Record<string, never>;
export declare const useGroup: Flux.Use<RetrieveGroupQuery, group.Group>;
//# sourceMappingURL=queries.d.ts.map