import { lineplot } from "@synnaxlabs/client";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type TimeRange, type TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type FrameProps } from "./Frame";
import { type LegendProps as BaseLegendProps } from "./Legend";
import { type measure } from "./measure/aether";
import { Range } from "./range";
import { type Viewport } from "../viewport";
export type ResolvedRange = {
    variant: "static";
    timeRange: TimeRange;
} | {
    variant: "dynamic";
    span: TimeSpan;
};
export declare const axisLabel: (key: lineplot.AxisKey) => string;
export interface LinePlotProps extends FrameProps {
    editable?: boolean;
    enableTriggers?: Triggers.Condition;
    resolvedRanges?: Map<string, ResolvedRange>;
    legendVariant?: BaseLegendProps["variant"];
    enableTooltip?: boolean;
    enableMeasure?: boolean;
    measureMode?: measure.Mode;
    onMeasureModeChange?: (mode: measure.Mode) => void;
    initialViewport?: Viewport.UseProps["initial"];
    onViewportChange?: Viewport.UseProps["onChange"];
    viewportTriggers?: Viewport.UseProps["triggers"];
    rangeProviderProps?: Range.ProviderProps;
    onSelectRule?: (key: string) => void;
    hiddenLines?: Set<string>;
    onLineVisibleChange?: (lineKey: string, visible: boolean) => void;
}
export declare const LinePlot: ({ editable, enableTriggers, resolvedRanges, legendVariant, enableTooltip, enableMeasure, measureMode, onMeasureModeChange, initialViewport, onViewportChange, viewportTriggers, rangeProviderProps, onSelectRule, hiddenLines, onLineVisibleChange, children, ref, ...rest }: LinePlotProps) => ReactElement;
//# sourceMappingURL=LinePlot.d.ts.map