import { observe } from "@synnaxlabs/x";
import { type BooleanSinkSpec, type NumberSinkSpec, type Sink, type StringSinkSpec, type Telem } from "../telem";
export declare class TestSink<V> extends observe.Observer<void> implements Sink<V>, Telem {
    static readonly TYPE = "test-sink";
    readonly id: string;
    values: V[];
    private _destructor;
    constructor();
    get lastValue(): V | undefined;
    set(...values: V[]): void;
    clear(): void;
    cleanup(): void;
}
export declare const sink: <V>() => TestSink<V>;
export declare const booleanSinkSpec: (sink: TestSink<boolean>) => BooleanSinkSpec;
export declare const numberSinkSpec: (sink: TestSink<number>) => NumberSinkSpec;
export declare const stringSinkSpec: (sink: TestSink<string>) => StringSinkSpec;
//# sourceMappingURL=sink.d.ts.map