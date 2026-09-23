import { Synnax } from "@synnaxlabs/client";
import { z } from "zod";
import { aether } from "../../aether/aether";
declare const stateZ: z.ZodObject<{
    props: z.ZodNullable<z.ZodObject<{
        host: z.ZodString;
        port: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
        username: z.ZodString;
        password: z.ZodString;
        connectivityPollFrequency: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
        clockSkewThreshold: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
        secure: z.ZodDefault<z.ZodBoolean>;
        name: z.ZodOptional<z.ZodString>;
        retry: z.ZodOptional<z.ZodObject<{
            baseInterval: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
            maxInterval: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
            maxRetries: z.ZodOptional<z.ZodNumber>;
            scale: z.ZodOptional<z.ZodNumber>;
            jitter: z.ZodOptional<z.ZodNumber>;
        }, z.core.$strip>>;
        cache: z.ZodDefault<z.ZodBoolean>;
        onInternalError: z.ZodOptional<z.ZodFunction<z.ZodTuple<[z.ZodCustom<Error, Error>], null>, z.ZodUnknown>>;
    }, z.core.$strip>>;
}, z.core.$strip>;
export interface ContextValue {
    client: Synnax | null;
}
export declare const ZERO_CONTEXT_VALUE: ContextValue;
export declare class Provider extends aether.Composite<typeof stateZ, ContextValue> {
    static readonly TYPE = "synnax.Provider";
    static readonly stateZ: z.ZodObject<{
        props: z.ZodNullable<z.ZodObject<{
            host: z.ZodString;
            port: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
            username: z.ZodString;
            password: z.ZodString;
            connectivityPollFrequency: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
            clockSkewThreshold: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
            secure: z.ZodDefault<z.ZodBoolean>;
            name: z.ZodOptional<z.ZodString>;
            retry: z.ZodOptional<z.ZodObject<{
                baseInterval: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
                maxInterval: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
                maxRetries: z.ZodOptional<z.ZodNumber>;
                scale: z.ZodOptional<z.ZodNumber>;
                jitter: z.ZodOptional<z.ZodNumber>;
            }, z.core.$strip>>;
            cache: z.ZodDefault<z.ZodBoolean>;
            onInternalError: z.ZodOptional<z.ZodFunction<z.ZodTuple<[z.ZodCustom<Error, Error>], null>, z.ZodUnknown>>;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        props: z.ZodNullable<z.ZodObject<{
            host: z.ZodString;
            port: z.ZodUnion<[z.ZodNumber, z.ZodString]>;
            username: z.ZodString;
            password: z.ZodString;
            connectivityPollFrequency: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
            clockSkewThreshold: z.ZodDefault<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                value: z.ZodBigInt;
            }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                value: bigint;
            }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
            secure: z.ZodDefault<z.ZodBoolean>;
            name: z.ZodOptional<z.ZodString>;
            retry: z.ZodOptional<z.ZodObject<{
                baseInterval: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
                maxInterval: z.ZodOptional<z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
                    value: z.ZodBigInt;
                }, z.core.$strip>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, {
                    value: bigint;
                }>>, z.ZodPipe<z.ZodString, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, bigint>>, z.ZodCustom<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<import("@synnaxlabs/x").TimeSpan, import("@synnaxlabs/x").Rate>>]>>;
                maxRetries: z.ZodOptional<z.ZodNumber>;
                scale: z.ZodOptional<z.ZodNumber>;
                jitter: z.ZodOptional<z.ZodNumber>;
            }, z.core.$strip>>;
            cache: z.ZodDefault<z.ZodBoolean>;
            onInternalError: z.ZodOptional<z.ZodFunction<z.ZodTuple<[z.ZodCustom<Error, Error>], null>, z.ZodUnknown>>;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    private closeClient;
}
export declare const use: (ctx: aether.Context) => Synnax | null;
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=provider.d.ts.map