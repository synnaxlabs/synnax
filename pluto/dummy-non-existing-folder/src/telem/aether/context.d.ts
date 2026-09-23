import { type destructor, type observe } from "@synnaxlabs/x";
import { type aether } from "../../aether/aether";
import { CompoundFactory, type CreateOptions, type Factory } from "./factory";
import { type Sink, type Source, type Spec } from "./telem";
/** Provides utilities for creating and managing telemetry sources and sinks. */
export declare class Context {
    private factory;
    readonly key: string;
    readonly parent?: Context;
    constructor(factory: CompoundFactory, parent?: Context);
    child(factories: Factory | Factory[], parent?: Context): Context;
    create<T>(spec: Spec, options?: CreateOptions): T;
}
export declare const CONTEXT_KEY = "pluto-telem-context";
export declare const useContext: (ctx: aether.Context) => Context;
export declare const setContext: (ctx: aether.Context, prov: Context) => void;
export declare const useChildContext: (ctx: aether.Context, factories: Factory | Factory[], prev: Context) => Context;
declare class Memoized<V> {
    private readonly spec;
    readonly wrapped: V;
    private readonly prevProv;
    constructor(wrapped: V, prevProv: Context, prevSpec: Spec);
    shouldUpdate(prov: Context, spec: Spec): boolean;
}
export declare class MemoizedSource<V, S extends Source<V> = Source<V>> extends Memoized<S> {
    value(): V;
    loading(): boolean;
    cleanup(): void;
    onChange(handler: observe.Handler<void>): destructor.Destructor;
}
declare class MemoizedSink<V> extends Memoized<Sink<V>> {
    set(...values: V[]): void;
    cleanup(): void;
}
export declare const useSource: <V, S extends Source<V> = Source<V>>(ctx: aether.Context, spec: Spec, prev: S | MemoizedSource<V, S>, options?: CreateOptions) => MemoizedSource<V, S>;
export declare const useSink: <V>(ctx: aether.Context, spec: Spec, prev: Sink<V>, options?: CreateOptions) => MemoizedSink<V>;
export {};
//# sourceMappingURL=context.d.ts.map