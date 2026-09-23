import { type theme } from "@synnaxlabs/lyra/theme";
import { color, type destructor, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
/** Seconds without a sample before a source is considered stale. */
export declare const DEFAULT_TIMEOUT = 5;
/**
 * configZ carries the staleness config every source-backed component adds to its own
 * state. Components that resolve the stale color on the worker add stalenessColor
 * themselves; DOM-rendered ones read it from their config instead.
 */
export declare const configZ: z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
/**
 * stateZ adds the reported staleness, which crosses to the DOM on every transition.
 * Extend it only when the DOM half renders the stale state. A component that draws on
 * the worker extends configZ and keeps staleness in its internal state.
 */
export declare const stateZ: z.ZodObject<{
    stalenessTimeout: z.ZodDefault<z.ZodNumber>;
    stale: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
export interface EntryProps {
    /** Returns the current staleness timeout, in seconds. */
    timeout: () => number;
    /** Returns the staleness the source currently reports. */
    stale: () => boolean;
    /** Receives each staleness transition. */
    onChange: (stale: boolean) => void;
}
export interface Registration {
    /** Records an arriving sample. This clears staleness and restarts the countdown. */
    received: () => void;
    /** Releases the registration. */
    cleanup: destructor.Destructor;
    /** The source this registration counts arrivals for. Arrival state belongs to one
     * source, so a caller holding a registration for a different one must replace it. */
    readonly source: unknown;
}
declare const providerStateZ: z.ZodObject<{
    sweepInterval: z.ZodUnion<readonly [z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
        value: z.ZodBigInt;
    }, z.core.$strip>, z.ZodTransform<TimeSpan, {
        value: bigint;
    }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>]>;
}, z.core.$strip>;
/**
 * Provider turns a registered source stale when no sample arrives within its timeout.
 * Staleness measures arrival, not sample time. It answers "is this source still
 * sending", not "is the newest sample recent". A source that delivers old data keeps
 * reading live, and a source that sends more slowly than its timeout reads stale even
 * while it is healthy. Give a source that sends on change a timeout longer than the
 * longest gap you expect between changes. One periodic sweep serves every source below
 * the Provider, so the cost stays flat as sources and sample rates grow. The sweep
 * compares against the monotonic clock, so a throttled or suspended worker resolves to
 * the correct state when it wakes.
 */
export declare class Provider extends aether.Composite<typeof providerStateZ> {
    static readonly TYPE = "staleness.Provider";
    static readonly z: z.ZodObject<{
        sweepInterval: z.ZodUnion<readonly [z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>]>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        sweepInterval: z.ZodUnion<readonly [z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodUnion<readonly [z.ZodPipe<z.ZodObject<{
            value: z.ZodBigInt;
        }, z.core.$strip>, z.ZodTransform<TimeSpan, {
            value: bigint;
        }>>, z.ZodPipe<z.ZodString, z.ZodTransform<TimeSpan, string>>, z.ZodPipe<z.ZodNumber, z.ZodTransform<TimeSpan, number>>, z.ZodPipe<z.ZodBigInt, z.ZodTransform<TimeSpan, bigint>>, z.ZodCustom<TimeSpan, TimeSpan>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").TimeStamp, import("@synnaxlabs/x").TimeStamp>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").TimeStamp>>, z.ZodPipe<z.ZodCustom<import("@synnaxlabs/x").Rate, import("@synnaxlabs/x").Rate>, z.ZodTransform<TimeSpan, import("@synnaxlabs/x").Rate>>]>]>;
    }, z.core.$strip>;
    private readonly entries;
    private interval?;
    private sweepInterval;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    /** @returns a registration for `source`. Cleanup releases it. */
    register(props: EntryProps, source: unknown): Registration;
    private updateSweepInterval;
    private start;
    private stop;
    private release;
    private sweep;
}
/**
 * Registers `source` with the nearest {@link Provider}, reusing `prev` when it already
 * covers that source. A source swap gets a new registration, because arrival state
 * belongs to the source that produced it: carrying it over reports a channel stale that
 * the caller has never read a sample from.
 * @param prev - The registration returned by an earlier call, if any. A reused
 * registration keeps the props it was created with, so pass accessors that read live
 * values rather than props captured on this call.
 * @throws {NotFoundError} if no {@link Provider} is mounted above the caller.
 */
export declare const useRegistration: (ctx: aether.Context, prev: Registration | undefined, props: EntryProps, source: unknown) => Registration;
/** The leaf surface {@link useStateRegistration} drives. */
interface StatefulLeaf<S extends z.infer<typeof stateZ>> {
    readonly state: S;
    setState: (next: (prev: S) => S) => void;
}
/**
 * Registers a source that reports staleness through its aether state, where the DOM
 * half reads it. See {@link useRegistration} for reuse semantics.
 */
export declare const useStateRegistration: <S extends z.infer<typeof stateZ>>(ctx: aether.Context, prev: Registration | undefined, leaf: StatefulLeaf<S>, source: unknown) => Registration;
/** The leaf surface {@link useInternalRegistration} drives. */
interface InternalLeaf {
    readonly state: z.infer<typeof configZ>;
    readonly internal: {
        stale: boolean;
    };
}
/**
 * Registers a source that keeps staleness in `internal.stale`, off the state that
 * crosses to the DOM. See {@link useRegistration} for reuse semantics.
 * @param onTransition - Runs after each transition. A component that draws itself must
 * ask for a repaint here: with the source quiet, nothing else asks the canvas to
 * redraw.
 */
export declare const useInternalRegistration: (ctx: aether.Context, prev: Registration | undefined, leaf: InternalLeaf, source: unknown, onTransition: () => void) => Registration;
/**
 * Resolves the color that stale content renders in.
 * @param c - The configured staleness color. An absent color resolves to the theme's
 * warning shade.
 */
export declare const resolveColor: (c: color.Crude | undefined, theme: theme.Theme) => color.Color;
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=staleness.d.ts.map