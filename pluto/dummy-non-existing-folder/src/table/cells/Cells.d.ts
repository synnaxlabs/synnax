import "./Cells.css";
import { table } from "@synnaxlabs/client";
import { type border, box } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export declare const textConfigZ: import("zod").ZodObject<{
    variant: import("zod").ZodLiteral<"text">;
    value: import("zod").ZodDefault<import("zod").ZodString>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    weight: import("zod").ZodDefault<import("zod").ZodNumber>;
    align: import("zod").ZodDefault<import("zod").ZodEnum<{
        center: "center";
        end: "end";
        start: "start";
        stretch: "stretch";
    }>>;
    backgroundColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
    }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
}, import("zod/v4/core").$strip>;
export type TextConfig = table.TextCellConfig;
export type CellProps<C extends table.CellConfig = table.CellConfig> = C & {
    cellKey: string;
    box: box.Box;
    /**
     * Rounding of the cell's corners in px. Only cells at the table's outer corners
     * round.
     */
    borderRadius?: border.CrudeRadius;
    selected: boolean;
    editable: boolean;
    onSelect: (key: string, ev: React.MouseEvent) => void;
    onChange: (config: C) => void;
};
export declare const Text: ({ cellKey, onChange, value, selected, editable, onSelect, box: b, align, level, weight, backgroundColor, }: CellProps<TextConfig>) => ReactElement;
export declare const valueConfigZ: import("zod").ZodObject<{
    variant: import("zod").ZodLiteral<"value">;
    channel: import("zod").ZodDefault<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    rollingAverage: import("zod").ZodDefault<import("zod").ZodInt32>;
    precision: import("zod").ZodOptional<import("zod").ZodInt32>;
    notation: import("zod").ZodDefault<import("zod").ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
    redline: import("zod").ZodOptional<import("zod").ZodObject<{
        bounds: import("zod").ZodObject<{
            lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
            upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        }, import("zod/v4/core").$strip>;
        gradient: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
            key: import("zod").ZodString;
            color: import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
                rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
            }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
                r: import("zod").ZodInt;
                g: import("zod").ZodInt;
                b: import("zod").ZodInt;
                a: import("zod").ZodNumber;
            }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
                rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
            }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
                r: import("zod").ZodInt;
                g: import("zod").ZodInt;
                b: import("zod").ZodInt;
                a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
            }, import("zod/v4/core").$strip>]>;
            position: import("zod").ZodNumber;
            switched: import("zod").ZodOptional<import("zod").ZodBoolean>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
    }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
    units: import("zod").ZodDefault<import("zod").ZodString>;
    stalenessTimeout: import("zod").ZodDefault<import("zod").ZodNumber>;
    stalenessColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
    }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
}, import("zod/v4/core").$strip>;
export type ValueConfig = table.ValueCellConfig;
export declare const Value: ({ cellKey, channel, rollingAverage, precision, notation, borderRadius, level, color: textColor, redline, selected, box: b, onSelect, stalenessTimeout, stalenessColor, }: CellProps<ValueConfig>) => import("react").JSX.Element;
//# sourceMappingURL=Cells.d.ts.map