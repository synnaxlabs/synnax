import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { type Factory } from "./factory";
import { AbstractSink, AbstractSource, type Sink, type SinkSpec, type SinkTransformer, type Source, type SourceSpec, type SourceTransformer, type Spec, type Telem } from "./telem";
export declare const connectionZ: z.ZodObject<{
    from: z.ZodString;
    to: z.ZodString;
}, z.core.$strip>;
export type Connection = z.infer<typeof connectionZ>;
export declare const sourcePipelinePropsZ: z.ZodObject<{
    connections: z.ZodArray<z.ZodObject<{
        from: z.ZodString;
        to: z.ZodString;
    }, z.core.$strip>>;
    outlet: z.ZodString;
    segments: z.ZodRecord<z.ZodString, z.ZodObject<{
        type: z.ZodString;
        valueType: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
    }, z.core.$strip>>;
}, z.core.$strip>;
export type SourcePipelineProps = z.infer<typeof sourcePipelinePropsZ>;
export declare class PipelineFactory implements Factory {
    type: string;
    factory: Factory;
    constructor(factory: Factory);
    create(spec: Spec): Telem | null;
}
export declare class SourcePipeline<V> extends AbstractSource<typeof sourcePipelinePropsZ> implements Source<V> {
    static readonly TYPE = "source-pipeline";
    schema: z.ZodObject<{
        connections: z.ZodArray<z.ZodObject<{
            from: z.ZodString;
            to: z.ZodString;
        }, z.core.$strip>>;
        outlet: z.ZodString;
        segments: z.ZodRecord<z.ZodString, z.ZodObject<{
            type: z.ZodString;
            valueType: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    sources: Record<string, Source<any> | SourceTransformer<any, any>>;
    private get outlet();
    constructor(props: unknown, factory: Factory);
    value(): V;
    onChange(handler: () => void): destructor.Destructor;
    cleanup(): void;
}
export declare const sourcePipeline: <V extends string>(valueType: V, props: SourcePipelineProps) => SourceSpec<V>;
export declare const sinkPipelinePropsZ: z.ZodObject<{
    connections: z.ZodArray<z.ZodObject<{
        from: z.ZodString;
        to: z.ZodString;
    }, z.core.$strip>>;
    inlet: z.ZodString;
    segments: z.ZodRecord<z.ZodString, z.ZodObject<{
        type: z.ZodString;
        valueType: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"sink">;
    }, z.core.$strip>>;
}, z.core.$strip>;
export type SinkPipelineProps = z.infer<typeof sinkPipelinePropsZ>;
export declare class SinkPipeline<V> extends AbstractSink<typeof sinkPipelinePropsZ> implements Sink<V> {
    static readonly TYPE = "sink-pipeline";
    schema: z.ZodObject<{
        connections: z.ZodArray<z.ZodObject<{
            from: z.ZodString;
            to: z.ZodString;
        }, z.core.$strip>>;
        inlet: z.ZodString;
        segments: z.ZodRecord<z.ZodString, z.ZodObject<{
            type: z.ZodString;
            valueType: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"sink">;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    sinks: Record<string, Sink<any> | SinkTransformer<any, any>>;
    private get inlet();
    constructor(props: unknown, factory: Factory);
    set(...values: V[]): void;
    cleanup(): void;
}
export declare const sinkPipeline: <V extends string>(valueType: V, props: SinkPipelineProps) => SinkSpec<V>;
//# sourceMappingURL=pipeline.d.ts.map