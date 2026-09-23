import { theme } from "@synnaxlabs/lyra/theme";
import { box, color, type dimensions, direction, location, type text, xy } from "@synnaxlabs/x";
import { type FillTextOptions, type SugaredOffscreenCanvasRenderingContext2D } from "./canvas";
export interface Draw2DLineProps {
    stroke: color.Color;
    lineWidth: number;
    lineDash: number;
    start: xy.XY;
    end: xy.XY;
}
export interface Draw2DRuleProps extends Omit<Draw2DLineProps, "start" | "end"> {
    direction: direction.Direction;
    region: box.Box;
    position: number;
}
export interface Draw2DCircleProps {
    fill?: color.Color;
    stroke?: color.Color;
    strokeWidth?: number;
    lineDash?: number;
    radius: number | {
        inner: number;
        outer: number;
    };
    position: xy.XY;
    angle?: {
        lower: number;
        upper: number;
    };
    lineCap?: CanvasLineCap;
}
export interface Draw2DContainerProps {
    region: box.Box;
    bordered?: boolean | location.Outer | location.Outer[];
    rounded?: boolean;
    borderColor?: ColorSpec;
    borderRadius?: number;
    borderWidth?: number;
    backgroundColor?: ColorSpec;
}
export interface DrawTextProps extends FillTextOptions {
    text: string;
    position: xy.XY;
    level: text.Level;
    justify?: CanvasTextAlign;
    /**
     * Vertical placement. The canvas baselines place the em box, which the engines
     * disagree on, so "center" centers the ink instead, for glyphs sitting on the
     * baseline.
     */
    align?: CanvasTextBaseline | "center";
    weight?: text.Weight;
    shade?: theme.Shade;
    maxWidth?: number;
    code?: boolean;
    color?: ColorSpec;
}
export interface DrawTextInCenterProps extends Omit<DrawTextProps, "position" | "direction"> {
    box: box.Box;
}
export interface Draw2DMeasureTextContainerProps {
    text: string[];
    direction: direction.Direction;
    level: text.Level;
    spacing?: number;
}
export interface Draw2DBorderProps {
    region: box.Box;
    color?: ColorSpec;
    width?: number;
    radius?: number;
    location?: true | location.Outer | location.Outer[];
}
export interface Draw2DTextContainerProps extends Omit<Draw2DContainerProps, "region">, Draw2DMeasureTextContainerProps {
    position: xy.XY;
    offset?: xy.XY;
    root?: location.Corner;
}
export interface DrawList {
    length: number;
    position: xy.XY;
    itemHeight: number;
    spacing?: number;
    width: number;
    draw: (index: number, box: box.Box) => void;
    root?: location.Corner;
    offset?: xy.XY;
    padding?: xy.XY;
}
type ColorSpec = color.Crude | ((t: theme.Theme) => color.Color);
export declare class Draw2D {
    readonly canvas: SugaredOffscreenCanvasRenderingContext2D;
    readonly theme: theme.Theme;
    private charWidthCache;
    constructor(canvas: SugaredOffscreenCanvasRenderingContext2D, theme: theme.Theme);
    rule({ direction, region, position, ...rest }: Draw2DRuleProps): void;
    line({ stroke, lineWidth, lineDash, start, end }: Draw2DLineProps): void;
    circle({ fill, stroke, strokeWidth, lineDash, radius, position, angle, lineCap, }: Draw2DCircleProps): void;
    resolveColor(c: ColorSpec | undefined, fallback: ColorSpec): color.Color;
    resolveColor(c: ColorSpec): color.Color;
    border({ region, color: colorVal, width, radius, location, }: Draw2DBorderProps): void;
    container({ region, bordered, rounded, borderColor, borderRadius, borderWidth, backgroundColor, }: Draw2DContainerProps): void;
    textContainer(props: Draw2DTextContainerProps): void;
    spacedTextDrawF({ text: labels, direction: d, spacing, level, }: Draw2DMeasureTextContainerProps): [dimensions.Dimensions, (base: xy.XY) => void];
    list({ length, itemHeight, width, spacing, position, draw, root, offset, padding, }: DrawList): void;
    /**
     * Vertical offset that centers text ink within a rowHeight-tall band when the
     * text is drawn at the band's top with align "top". Canvas positions the em
     * box, not the ink, which sits a few pixels lower.
     */
    measureInkOffsetY(level: text.Level, rowHeight: number): number;
    measureCharWidth(level: text.Level): number;
    drawTextInCenter({ box: b, text: label, level }: DrawTextInCenterProps): void;
    text({ text, position, level, weight, shade, maxWidth, code, justify, align, useAtlas, color: colorVal, }: DrawTextProps): void;
}
export {};
//# sourceMappingURL=index.d.ts.map