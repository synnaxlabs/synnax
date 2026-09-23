import { lineplot, type project } from "@synnaxlabs/client";
import { color, type require } from "@synnaxlabs/x";
import { Flux } from "../flux";
export type RetrieveQuery = lineplot.RetrieveSingleParams;
export declare const use: Flux.Use<{
    key: string;
}, lineplot.LinePlot>, useEnsure: Flux.UseEnsure<{
    key: string;
}>, useTombstone: Flux.UseTombstone<{
    key: string;
}>, createSelector: Flux.CreateSelector<{
    key: string;
}, lineplot.LinePlot>;
export interface KeyParams {
    key: lineplot.Key;
}
export declare const useName: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => string;
export declare const useTitle: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    level: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
    visible: boolean;
};
export declare const useLegend: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    hidden: boolean;
    position: {
        x: number;
        y: number;
        root: {
            x: "left" | "right";
            y: "bottom" | "top";
        };
        units: {
            x: "decimal" | "px";
            y: "decimal" | "px";
        };
    };
};
export declare const useRanges: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    x1: string[];
    x2: string[];
    custom?: {
        variant: "dynamic";
        span: number;
    } | {
        variant: "static";
        start: string;
        end: string;
    } | undefined;
};
export declare const useAxes: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    x1: {
        key: "x1" | "x2" | "y1" | "y2" | "y3" | "y4";
        label: string;
        labelDirection: "x" | "y";
        labelLevel: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        bounds: {
            lower: number;
            upper: number;
        };
        manualBounds: {
            lower: boolean;
            upper: boolean;
        };
        tickSpacing: number;
        type?: "linear" | "time" | undefined;
    };
    x2: {
        key: "x1" | "x2" | "y1" | "y2" | "y3" | "y4";
        label: string;
        labelDirection: "x" | "y";
        labelLevel: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        bounds: {
            lower: number;
            upper: number;
        };
        manualBounds: {
            lower: boolean;
            upper: boolean;
        };
        tickSpacing: number;
        type?: "linear" | "time" | undefined;
    };
    y1: {
        key: "x1" | "x2" | "y1" | "y2" | "y3" | "y4";
        label: string;
        labelDirection: "x" | "y";
        labelLevel: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        bounds: {
            lower: number;
            upper: number;
        };
        manualBounds: {
            lower: boolean;
            upper: boolean;
        };
        tickSpacing: number;
        type?: "linear" | "time" | undefined;
    };
    y2: {
        key: "x1" | "x2" | "y1" | "y2" | "y3" | "y4";
        label: string;
        labelDirection: "x" | "y";
        labelLevel: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        bounds: {
            lower: number;
            upper: number;
        };
        manualBounds: {
            lower: boolean;
            upper: boolean;
        };
        tickSpacing: number;
        type?: "linear" | "time" | undefined;
    };
    y3: {
        key: "x1" | "x2" | "y1" | "y2" | "y3" | "y4";
        label: string;
        labelDirection: "x" | "y";
        labelLevel: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        bounds: {
            lower: number;
            upper: number;
        };
        manualBounds: {
            lower: boolean;
            upper: boolean;
        };
        tickSpacing: number;
        type?: "linear" | "time" | undefined;
    };
    y4: {
        key: "x1" | "x2" | "y1" | "y2" | "y3" | "y4";
        label: string;
        labelDirection: "x" | "y";
        labelLevel: "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
        bounds: {
            lower: number;
            upper: number;
        };
        manualBounds: {
            lower: boolean;
            upper: boolean;
        };
        tickSpacing: number;
        type?: "linear" | "time" | undefined;
    };
};
export declare const useXAxisKeys: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => ("x1" | "x2")[];
export declare const useYAxisKeys: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => ("y1" | "y2" | "y3" | "y4")[];
export declare const useAxisKeys: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => ("x1" | "x2" | "y1" | "y2" | "y3" | "y4")[];
export interface AxisParams {
    key: lineplot.Key;
    axisKey: lineplot.AxisKey;
}
export declare const useAxis: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<AxisParams, "key">) => lineplot.Axis;
export interface RawDerivedLine extends lineplot.Line, lineplot.LineKeyParts {
}
export interface DerivedLine extends Omit<RawDerivedLine, "color"> {
    color: color.Color;
    isDefaultLabel: boolean;
}
export declare const useLines: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<KeyParams, "key"> | undefined) => DerivedLine[];
export declare const useLineKeys: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => string[];
export declare const useLineCount: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => number;
export interface YAxisParams {
    key: lineplot.Key;
    axisKey: lineplot.YAxisKey;
}
export interface XAxisParams {
    key: lineplot.Key;
    axisKey: lineplot.XAxisKey;
}
export declare const useYAxisChannels: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<YAxisParams, "key">) => number[];
export declare const useXAxisChannel: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<XAxisParams, "key">) => number;
export declare const useXAxisRanges: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<XAxisParams, "key">) => string[];
export declare const useCustomRange: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    variant: "dynamic";
    span: number;
} | {
    variant: "static";
    start: string;
    end: string;
} | undefined;
export declare const useXAxis: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<XAxisParams, "key">) => lineplot.Axis;
interface YAxisReturn {
    axis: lineplot.Axis;
    channels: lineplot.Channels[lineplot.YAxisKey];
    lineKeys: string[];
}
export declare const useYAxis: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<YAxisParams, "key">) => YAxisReturn;
export interface LineParams {
    key: lineplot.Key;
    lineKey: string;
}
export declare const useLine: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<LineParams, "key">) => require.Require<DerivedLine, "label">;
export interface DerivedRule extends Omit<lineplot.Rule, "color"> {
    color: color.Color;
}
export declare const useRules: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<KeyParams, "key"> | undefined) => DerivedRule[];
export interface RuleParams {
    key: lineplot.Key;
    ruleKey: string;
}
export declare const useRule: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<RuleParams, "key">) => DerivedRule;
export interface AxisRulesParams {
    key: lineplot.Key;
    axisKey: lineplot.AxisKey;
}
export declare const useAxisRuleKeys: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<AxisRulesParams, "key">) => string[];
export type UseDeleteParams = lineplot.Key | lineplot.Key[];
export declare const useDelete: Flux.UseUpdate<UseDeleteParams, UseDeleteParams, import("zod").ZodNever>;
export interface CreateParams extends lineplot.New {
    project?: project.Key;
}
export declare const useCreate: Flux.UseUpdate<CreateParams, lineplot.LinePlot, import("zod").ZodNever>;
export interface RenameParams extends Pick<lineplot.LinePlot, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, import("zod").ZodNever>;
export declare const useDispatch: () => Flux.UseDispatchReturn<string, lineplot.Action>, useUndoBase: (params: {
    key: string;
}) => Flux.UseUndoReturn, useRedoBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => Flux.UseRedoReturn, useSingleDispatchBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => (action: lineplot.Action[] | lineplot.Action) => void;
export declare const useSingleDispatch: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => (action: lineplot.Action[] | lineplot.Action) => void;
export declare const useUndo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Flux.UseUndoReturn;
export declare const useRedo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => Flux.UseRedoReturn;
export {};
//# sourceMappingURL=queries.d.ts.map