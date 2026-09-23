import { type Instrumentation } from "@synnaxlabs/alamos";
import { channel, control, type framer, status as cstatus, type Synnax } from "@synnaxlabs/client";
import { type theme } from "@synnaxlabs/lyra/theme";
import { type CrudeSeries } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { status } from "../../../status/aether";
import { telem } from "../../aether";
import { AbstractSink } from "../../aether/telem";
import { Colors } from "./colors";
export declare const statusZ: z.ZodEnum<{
    acquired: "acquired";
    failed: "failed";
    overridden: "overridden";
    released: "released";
}>;
export type Status = z.infer<typeof statusZ>;
export declare const controllerStateZ: z.ZodObject<{
    name: z.ZodString;
    authority: z.ZodDefault<z.ZodNumber>;
    status: z.ZodOptional<z.ZodEnum<{
        acquired: "acquired";
        failed: "failed";
        overridden: "overridden";
        released: "released";
    }>>;
    needsControlOf: z.ZodDefault<z.ZodArray<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>>;
    disabled: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
export declare const controllerMethodsZ: {
    acquire: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    release: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
};
/**
 * Opens the writer a {@link Controller} commands through. The client is an argument
 * rather than a binding because a Controller resolves its own only after construction.
 */
export type OpenWriter = (client: Synnax, config: framer.WriterConfig) => Promise<framer.Writer>;
interface InternalState {
    client: Synnax | null;
    instrumentation: Instrumentation;
    colors: Colors;
    addStatus: status.Adder;
    runAsync: status.ErrorHandler;
    theme: theme.Theme;
    telemCtx: telem.Context;
}
interface AetherControllerTelem extends telem.Telem {
    needsControlOf: (client: Synnax) => Promise<channel.Key[]>;
}
/**
 * @summary Acquires control over a set of channels by opening a writer to a Synnax
 * cluster, and then acts as a factory for telemetry that can be used to send commands
 * to that writer.
 */
export declare class Controller extends aether.Composite<typeof controllerStateZ, InternalState, aether.Component, typeof controllerMethodsZ> implements telem.Factory, aether.HandlersFromSchema<typeof controllerMethodsZ> {
    static readonly TYPE = "Controller";
    static readonly METHODS: {
        acquire: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
        release: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    };
    schema: z.ZodObject<{
        name: z.ZodString;
        authority: z.ZodDefault<z.ZodNumber>;
        status: z.ZodOptional<z.ZodEnum<{
            acquired: "acquired";
            failed: "failed";
            overridden: "overridden";
            released: "released";
        }>>;
        needsControlOf: z.ZodDefault<z.ZodArray<z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>>>;
        disabled: z.ZodDefault<z.ZodBoolean>;
    }, z.core.$strip>;
    methods: {
        acquire: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
        release: z.ZodFunction<z.ZodTuple<[], null>, z.ZodVoid>;
    };
    private readonly registry;
    private readonly openWriter;
    private writer?;
    private acquirePromise?;
    /** Set while an acquisition the user asked for is in flight. The disabled bar does
     * not apply to it: taking control is what clears the bar. */
    private acquireExplicit;
    constructor(props: aether.ComponentConstructorProps, openWriter?: OpenWriter);
    afterUpdate(ctx: aether.Context): void;
    /** The cluster this controller is bound to, or null while disconnected. */
    get client(): Synnax | null;
    afterDelete(): void;
    private updateNeedsControlOf;
    acquire(): void;
    release(): void;
    private doAcquire;
    private doAcquireImpl;
    private doRelease;
    private static isRetryable;
    private closeWriter;
    private withRetry;
    set(frame: framer.CrudeFrame | Record<channel.Key | channel.Name, CrudeSeries>): Promise<void>;
    setAuthority(channels: channel.Key[], value: control.Authority): Promise<void>;
    releaseAuthority(keys: channel.Key[]): Promise<void>;
    deleteTelem(t: AetherControllerTelem): void;
    /** @implements telem.Factory to create telemetry that is bound to this controller. */
    create<T>(spec: telem.Spec): T | null;
}
export declare const setChannelValuePropsZ: z.ZodObject<{
    channel: z.ZodNumber;
}, z.core.$strip>;
export type SetChannelValueProps = z.infer<typeof setChannelValuePropsZ>;
export declare class SetChannelValue extends AbstractSink<typeof setChannelValuePropsZ> implements telem.NumberSink, AetherControllerTelem {
    static readonly TYPE = "controlled-numeric-telem-sink";
    private readonly controller;
    private readonly runAsync;
    schema: z.ZodObject<{
        channel: z.ZodNumber;
    }, z.core.$strip>;
    constructor(controller: Controller, runAsync: status.ErrorHandler, props: unknown);
    invalidate(): void;
    cleanup(): void;
    needsControlOf(client: Synnax): Promise<channel.Key[]>;
    set(...values: number[]): void;
}
export declare const setChannelValue: (props: SetChannelValueProps) => telem.NumberSinkSpec;
export declare const acquireChannelControlPropsZ: z.ZodObject<{
    authority: z.ZodDefault<z.ZodNumber>;
    channel: z.ZodNumber;
}, z.core.$strip>;
export type AcquireChannelControlProps = z.infer<typeof acquireChannelControlPropsZ>;
export declare class AcquireChannelControl extends AbstractSink<typeof acquireChannelControlPropsZ> implements telem.BooleanSink, AetherControllerTelem {
    static readonly TYPE = "acquire-channel-control";
    private readonly controller;
    private readonly runAsync;
    schema: z.ZodObject<{
        authority: z.ZodDefault<z.ZodNumber>;
        channel: z.ZodNumber;
    }, z.core.$strip>;
    constructor(controller: Controller, runAsync: status.ErrorHandler, props: unknown);
    cleanup(): void;
    needsControlOf(client: Synnax): Promise<channel.Key[]>;
    set(acquire: boolean): void;
}
export declare const acquireChannelControl: (props: AcquireChannelControlProps) => telem.BooleanSinkSpec;
export declare const authoritySourceProps: z.ZodObject<{
    channel: z.ZodNumber;
}, z.core.$strip>;
export type AuthoritySourceProps = z.infer<typeof authoritySourceProps>;
export declare const authoritySourceDetailsZ: z.ZodObject<{
    valid: z.ZodBoolean;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
    authority: z.ZodNumber;
}, z.core.$strip>;
export type AuthoritySourceDetails = z.infer<typeof authoritySourceDetailsZ>;
export declare class AuthoritySource extends telem.AbstractSource<typeof authoritySourceProps> implements telem.StatusSource<typeof authoritySourceDetailsZ>, AetherControllerTelem {
    static readonly TYPE = "controlled-status-source";
    private readonly colors;
    private readonly controller;
    private readonly retrieve;
    private readonly stopListening;
    schema: z.ZodObject<{
        channel: z.ZodNumber;
    }, z.core.$strip>;
    constructor(controller: Controller, colors: Colors, runAsync: status.ErrorHandler, props: unknown);
    needsControlOf(): Promise<channel.Key[]>;
    value(): cstatus.Status<typeof authoritySourceDetailsZ>;
    cleanup(): void;
}
export declare const authoritySource: (props: AuthoritySourceProps) => telem.StatusSourceSpec;
export {};
//# sourceMappingURL=controller.d.ts.map