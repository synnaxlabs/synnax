import { type Factory } from "./factory";
import { type BooleanSinkSpec, type BooleanSourceSpec, type ColorSourceSpec, type NumberSinkSpec, type NumberSourceSpec, type SeriesSourceSpec, type Spec, type StatusSourceSpec, type StringSinkSpec, type StringSourceSpec, type Telem } from "./telem";
export declare const noopBooleanSinkSpec: BooleanSinkSpec;
export declare const noopNumericSinkSpec: NumberSinkSpec;
export declare const noopBooleanSourceSpec: BooleanSourceSpec;
export declare const noopNumericSourceSpec: NumberSourceSpec;
export declare const noopStringSinkSpec: StringSinkSpec;
export declare const noopStringSourceSpec: StringSourceSpec;
export declare const noopStatusSourceSpec: StatusSourceSpec;
export declare const noopColorSourceSpec: ColorSourceSpec;
export declare const noopSeriesSourceSpec: SeriesSourceSpec;
export declare class NoopFactory implements Factory {
    type: string;
    create(spec: Spec): Telem | null;
}
//# sourceMappingURL=noop.d.ts.map