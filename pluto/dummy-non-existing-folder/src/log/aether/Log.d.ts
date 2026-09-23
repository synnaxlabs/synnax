import { log } from "@synnaxlabs/client";
import { type theme } from "@synnaxlabs/lyra/theme";
import { color, type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../aether/aether";
import { type LogEntry, type LogSource } from "./telem/types";
import { telem } from "../../telem/aether";
import { Draw2D } from "../../vis/draw2d";
import { render } from "../../vis/render";
export declare const logStateZ: z.ZodObject<{
    channels: z.ZodDefault<z.ZodArray<z.ZodObject<{
        channel: z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>;
        color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        } | [number, number, number] | [number, number, number, number]>>;
        notation: z.ZodDefault<z.ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        precision: z.ZodDefault<z.ZodInt32>;
        alias: z.ZodDefault<z.ZodString>;
        timestamp: z.ZodPrefault<z.ZodObject<{
            format: z.ZodEnum<{
                ISO: "ISO";
                ISODate: "ISODate";
                date: "date";
                dateTime: "dateTime";
                preciseDate: "preciseDate";
                preciseTime: "preciseTime";
                time: "time";
            }>;
            tz: z.ZodEnum<{
                UTC: "UTC";
                local: "local";
            }>;
        }, z.core.$strip>>;
    }, z.core.$strip>>>;
    timestampPrecision: z.ZodDefault<z.ZodInt32>;
    channelNamesHidden: z.ZodDefault<z.ZodBoolean>;
    receiptTimestampHidden: z.ZodDefault<z.ZodBoolean>;
    region: z.ZodObject<{
        one: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        two: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        root: z.ZodObject<{
            x: z.ZodEnum<{
                left: "left";
                right: "right";
            }>;
            y: z.ZodEnum<{
                bottom: "bottom";
                top: "top";
            }>;
        }, z.core.$strip>;
    }, z.core.$strip>;
    wheelPos: z.ZodNumber;
    scrolling: z.ZodBoolean;
    resumedAt: z.ZodDefault<z.ZodNumber>;
    empty: z.ZodBoolean;
    visible: z.ZodBoolean;
    channelNames: z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodString>>;
    channelDataTypes: z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodString>>;
    telem: z.ZodDefault<z.ZodObject<{
        type: z.ZodString;
        props: z.ZodAny;
        transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
        variant: z.ZodLiteral<"source">;
        valueType: z.ZodLiteral<"log">;
    }, z.core.$strip>>;
    font: z.ZodDefault<z.ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    color: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
    overshoot: z.ZodDefault<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
    selectionStart: z.ZodDefault<z.ZodNumber>;
    selectionEnd: z.ZodDefault<z.ZodNumber>;
    visibleStart: z.ZodDefault<z.ZodNumber>;
    selectedText: z.ZodDefault<z.ZodString>;
    selectedLines: z.ZodDefault<z.ZodArray<z.ZodObject<{
        text: z.ZodString;
        color: z.ZodString;
    }, z.core.$strip>>>;
    computedLineHeight: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
interface InternalState {
    theme: theme.Theme;
    render: render.Context;
    draw2d: Draw2D;
    telem: telem.MemoizedSource<LogEntry[], LogSource>;
    configs: Record<string, log.ChannelEntry>;
    textColor: color.Color;
    prefixColors: Record<string, color.Color>;
    defaultPrefixColor: color.Color;
    valueColors: Record<string, color.Color>;
    displayNames: Record<string, string>;
    namePadding: Record<string, string>;
    charWidth: number;
    lineHeight: number;
    selectionOffsetY: number;
    tsLen: number;
    selectionColor: color.Color;
    stopListeningTelem?: destructor.Destructor;
}
interface ScrollbackState {
    offset: number;
    offsetRef: number;
    scrollRef: number;
    awayFromEnd: boolean;
}
export declare class Log extends aether.Leaf<typeof logStateZ, InternalState> {
    static readonly TYPE = "log";
    static readonly z: z.ZodObject<{
        channels: z.ZodDefault<z.ZodArray<z.ZodObject<{
            channel: z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>;
            color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
            } | [number, number, number] | [number, number, number, number]>>;
            notation: z.ZodDefault<z.ZodEnum<{
                engineering: "engineering";
                scientific: "scientific";
                standard: "standard";
            }>>;
            precision: z.ZodDefault<z.ZodInt32>;
            alias: z.ZodDefault<z.ZodString>;
            timestamp: z.ZodPrefault<z.ZodObject<{
                format: z.ZodEnum<{
                    ISO: "ISO";
                    ISODate: "ISODate";
                    date: "date";
                    dateTime: "dateTime";
                    preciseDate: "preciseDate";
                    preciseTime: "preciseTime";
                    time: "time";
                }>;
                tz: z.ZodEnum<{
                    UTC: "UTC";
                    local: "local";
                }>;
            }, z.core.$strip>>;
        }, z.core.$strip>>>;
        timestampPrecision: z.ZodDefault<z.ZodInt32>;
        channelNamesHidden: z.ZodDefault<z.ZodBoolean>;
        receiptTimestampHidden: z.ZodDefault<z.ZodBoolean>;
        region: z.ZodObject<{
            one: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            two: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            root: z.ZodObject<{
                x: z.ZodEnum<{
                    left: "left";
                    right: "right";
                }>;
                y: z.ZodEnum<{
                    bottom: "bottom";
                    top: "top";
                }>;
            }, z.core.$strip>;
        }, z.core.$strip>;
        wheelPos: z.ZodNumber;
        scrolling: z.ZodBoolean;
        resumedAt: z.ZodDefault<z.ZodNumber>;
        empty: z.ZodBoolean;
        visible: z.ZodBoolean;
        channelNames: z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodString>>;
        channelDataTypes: z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodString>>;
        telem: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"log">;
        }, z.core.$strip>>;
        font: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        color: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        overshoot: z.ZodDefault<z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>>;
        selectionStart: z.ZodDefault<z.ZodNumber>;
        selectionEnd: z.ZodDefault<z.ZodNumber>;
        visibleStart: z.ZodDefault<z.ZodNumber>;
        selectedText: z.ZodDefault<z.ZodString>;
        selectedLines: z.ZodDefault<z.ZodArray<z.ZodObject<{
            text: z.ZodString;
            color: z.ZodString;
        }, z.core.$strip>>>;
        computedLineHeight: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        channels: z.ZodDefault<z.ZodArray<z.ZodObject<{
            channel: z.ZodUnion<[z.ZodUInt32, z.ZodPipe<z.ZodString, z.ZodTransform<number, string>>]>;
            color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodNumber;
            }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
                rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
            }, z.core.$strip>, z.ZodObject<{
                r: z.ZodInt;
                g: z.ZodInt;
                b: z.ZodInt;
                a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
            }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
            } | [number, number, number] | [number, number, number, number]>>;
            notation: z.ZodDefault<z.ZodEnum<{
                engineering: "engineering";
                scientific: "scientific";
                standard: "standard";
            }>>;
            precision: z.ZodDefault<z.ZodInt32>;
            alias: z.ZodDefault<z.ZodString>;
            timestamp: z.ZodPrefault<z.ZodObject<{
                format: z.ZodEnum<{
                    ISO: "ISO";
                    ISODate: "ISODate";
                    date: "date";
                    dateTime: "dateTime";
                    preciseDate: "preciseDate";
                    preciseTime: "preciseTime";
                    time: "time";
                }>;
                tz: z.ZodEnum<{
                    UTC: "UTC";
                    local: "local";
                }>;
            }, z.core.$strip>>;
        }, z.core.$strip>>>;
        timestampPrecision: z.ZodDefault<z.ZodInt32>;
        channelNamesHidden: z.ZodDefault<z.ZodBoolean>;
        receiptTimestampHidden: z.ZodDefault<z.ZodBoolean>;
        region: z.ZodObject<{
            one: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            two: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            root: z.ZodObject<{
                x: z.ZodEnum<{
                    left: "left";
                    right: "right";
                }>;
                y: z.ZodEnum<{
                    bottom: "bottom";
                    top: "top";
                }>;
            }, z.core.$strip>;
        }, z.core.$strip>;
        wheelPos: z.ZodNumber;
        scrolling: z.ZodBoolean;
        resumedAt: z.ZodDefault<z.ZodNumber>;
        empty: z.ZodBoolean;
        visible: z.ZodBoolean;
        channelNames: z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodString>>;
        channelDataTypes: z.ZodDefault<z.ZodRecord<z.ZodString, z.ZodString>>;
        telem: z.ZodDefault<z.ZodObject<{
            type: z.ZodString;
            props: z.ZodAny;
            transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
            variant: z.ZodLiteral<"source">;
            valueType: z.ZodLiteral<"log">;
        }, z.core.$strip>>;
        font: z.ZodDefault<z.ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        color: z.ZodDefault<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
            rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
        }, z.core.$strip>, z.ZodObject<{
            r: z.ZodInt;
            g: z.ZodInt;
            b: z.ZodInt;
            a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
        }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
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
        overshoot: z.ZodDefault<z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>>;
        selectionStart: z.ZodDefault<z.ZodNumber>;
        selectionEnd: z.ZodDefault<z.ZodNumber>;
        visibleStart: z.ZodDefault<z.ZodNumber>;
        selectedText: z.ZodDefault<z.ZodString>;
        selectedLines: z.ZodDefault<z.ZodArray<z.ZodObject<{
            text: z.ZodString;
            color: z.ZodString;
        }, z.core.$strip>>>;
        computedLineHeight: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    entries: LogEntry[];
    scrollState: ScrollbackState;
    private renderKey;
    get lineHeight(): number;
    get totalHeight(): number;
    get visibleLineCount(): number;
    afterUpdate(ctx: aether.Context): void;
    private checkEmpty;
    afterDelete(): void;
    private requestRender;
    private calcVisibleLineCount;
    render(): render.Cleanup | undefined;
    private renderScrollbar;
    private clampSelection;
    private renderSelection;
    private formatEntry;
    private updateSelectedText;
    private renderElements;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=Log.d.ts.map