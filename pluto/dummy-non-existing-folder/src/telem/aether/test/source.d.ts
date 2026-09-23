import { type color, observe } from "@synnaxlabs/x";
import { type BooleanSourceSpec, type ColorSourceSpec, type NumberSourceSpec, type Source, type StringSourceSpec, type Telem } from "../telem";
export declare class TestSource<V> extends observe.Observer<void> implements Source<V>, Telem {
    static readonly TYPE = "test-source";
    readonly id: string;
    private _value;
    private _destructor;
    constructor(initialValue: V);
    value(): V;
    setValue(v: V): void;
    cleanup(): void;
}
export declare const source: <V>(initialValue: V) => TestSource<V>;
export declare const booleanSourceSpec: (source: TestSource<boolean>) => BooleanSourceSpec;
export declare const numberSourceSpec: (source: TestSource<number>) => NumberSourceSpec;
export declare const stringSourceSpec: (source: TestSource<string>) => StringSourceSpec;
export declare const colorSourceSpec: (source: TestSource<color.Color>) => ColorSourceSpec;
//# sourceMappingURL=source.d.ts.map