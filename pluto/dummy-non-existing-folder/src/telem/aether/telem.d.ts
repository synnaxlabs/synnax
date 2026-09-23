import { type status } from "@synnaxlabs/client";
import { type bounds, type color, type destructor, type MultiSeries, observe } from "@synnaxlabs/x";
import { z } from "zod";
/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export declare const specZ: z.ZodObject<{
    type: z.ZodString;
    variant: z.ZodEnum<{
        sink: "sink";
        "sink-transformer": "sink-transformer";
        source: "source";
        "source-transformer": "source-transformer";
    }>;
    valueType: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
}, z.core.$strip>;
export declare const sourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    valueType: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
}, z.core.$strip>;
export declare const sinkSpecZ: z.ZodObject<{
    type: z.ZodString;
    valueType: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"sink">;
}, z.core.$strip>;
export declare const sourceTransformerSpecZ: z.ZodObject<{
    type: z.ZodString;
    valueType: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source-transformer">;
}, z.core.$strip>;
export declare const sinkTransformerSpecZ: z.ZodObject<{
    type: z.ZodString;
    valueType: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"sink-transformer">;
}, z.core.$strip>;
/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export type Spec = z.infer<typeof specZ>;
export type SourceSpec<V extends string> = z.infer<typeof sourceSpecZ> & {
    valueType: V;
};
export type SinkSpec<V extends string> = z.infer<typeof sinkSpecZ> & {
    valueType: V;
};
export interface ValueProps {
    onLoad: () => void;
}
export interface Telem {
    cleanup?: () => void;
}
export interface Source<V> extends Telem, observe.Observable<void> {
    value: (props?: ValueProps) => V;
    /** @returns true while the source's initial read is in flight. */
    loading?: () => boolean;
}
export interface Sink<V> extends Telem {
    set: (...values: V[]) => void;
}
export interface SourceTransformer<I, O> extends Telem, Source<O> {
    setSources: (sources: Record<string, Source<I>>) => void;
}
export interface SinkTransformer<I, O> extends Telem, Sink<I> {
    setSinks: (sinks: Record<string, Sink<O>>) => void;
}
export type SeriesSource = Source<[bounds.Bounds, MultiSeries]>;
export declare const seriesSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"series">;
}, z.core.$strip>;
export type SeriesSourceSpec = z.infer<typeof seriesSourceSpecZ>;
export type BooleanSource = Source<boolean>;
export declare const booleanSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"boolean">;
}, z.core.$strip>;
export type BooleanSourceSpec = z.infer<typeof booleanSourceSpecZ>;
export type BooleanSink = Sink<boolean>;
export declare const booleanSinkSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"sink">;
    valueType: z.ZodLiteral<"boolean">;
}, z.core.$strip>;
export type BooleanSinkSpec = z.infer<typeof booleanSinkSpecZ>;
export type BooleanSinkTransformer = SinkTransformer<boolean, number>;
export declare const booleanSinkTransformerSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"sink-transformer">;
    valueType: z.ZodLiteral<"boolean">;
}, z.core.$strip>;
export type BooleanSinkTransformerSpec = z.infer<typeof booleanSinkTransformerSpecZ>;
export type BooleanSourceTransformer = SourceTransformer<number, boolean>;
export declare const booleanSourceTransformerSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source-transformer">;
    valueType: z.ZodLiteral<"boolean">;
}, z.core.$strip>;
export type BooleanSourceTransformerSpec = z.infer<typeof booleanSourceTransformerSpecZ>;
export type NumberSource = Source<number>;
export declare const numberSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"number">;
}, z.core.$strip>;
export type NumberSourceSpec = z.infer<typeof numberSourceSpecZ>;
export type NumberSink = Sink<number>;
export declare const numberSinkSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"sink">;
    valueType: z.ZodLiteral<"number">;
}, z.core.$strip>;
export type NumberSinkSpec = z.infer<typeof numberSinkSpecZ>;
export type ColorSource = Source<color.Color>;
export declare const colorSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"color">;
}, z.core.$strip>;
export type ColorSourceSpec = z.infer<typeof colorSourceSpecZ>;
export type StatusSource<Details extends z.ZodType = z.ZodNever> = Source<status.Status<Details>>;
export declare const statusSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"status">;
}, z.core.$strip>;
export type StatusSourceSpec = z.infer<typeof statusSourceSpecZ>;
export type StringSource = Source<string>;
export declare const stringSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"string">;
}, z.core.$strip>;
export type StringSourceSpec = z.infer<typeof stringSourceSpecZ>;
export type StringSink = Sink<string>;
export declare const stringSinkSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"sink">;
    valueType: z.ZodLiteral<"string">;
}, z.core.$strip>;
export type StringSinkSpec = z.infer<typeof stringSinkSpecZ>;
export declare abstract class Base<P extends z.ZodType> extends observe.BaseObserver<void> {
    private props_;
    private readonly uProps_;
    abstract schema: P;
    constructor(props: unknown);
    get props(): z.infer<P>;
    cleanup(): void;
}
export declare abstract class AbstractSource<P extends z.ZodType> extends Base<P> {
    protected loading_: boolean;
    loading(): boolean;
    protected declareLoaded(): void;
}
export declare abstract class AbstractSink<P extends z.ZodType> extends Base<P> {
}
export declare abstract class UnarySourceTransformer<I, O, P extends z.ZodType> extends AbstractSource<P> implements SourceTransformer<I, O> {
    source_: Source<I> | undefined;
    private get source();
    value(): O;
    onChange(handler: () => void): destructor.Destructor;
    setSources(sources: Record<string, Source<I>>): void;
    protected shouldNotify(_: I): boolean;
    protected abstract transform(_: I): O;
}
export declare abstract class MultiSourceTransformer<I, O, P extends z.ZodType> extends AbstractSource<P> implements SourceTransformer<I, O> {
    sources: Record<string, Source<I>>;
    value(): O;
    setSources(sources: Record<string, Source<I>>): void;
    protected abstract transform(_: Record<string, I>): O;
}
export declare abstract class UnarySinkTransformer<I, O, P extends z.ZodType> extends Base<P> implements SinkTransformer<I, O> {
    sinks: Record<string, Sink<O>>;
    private get sink();
    set(...values: I[]): void;
    setSinks(sinks: Record<string, Sink<O>>): void;
    protected abstract transform(..._: I[]): O[];
}
//# sourceMappingURL=telem.d.ts.map