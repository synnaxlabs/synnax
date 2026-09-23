import { type Instrumentation } from "@synnaxlabs/alamos";
import { type bounds } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../aether/aether";
import { XAxis } from "./XAxis";
import { measure } from "../measure/aether";
import { tooltip } from "../tooltip/aether";
import { status } from "../../status/aether";
import { type FindResult } from "../../vis/line/aether/line";
import { render } from "../../vis/render";
export type AxesBounds = Record<string, bounds.Bounds>;
export declare const linePlotStateZ: z.ZodObject<{
    container: z.ZodObject<{
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
    viewport: z.ZodObject<{
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
    hold: z.ZodDefault<z.ZodBoolean>;
    grid: z.ZodRecord<z.ZodString, z.ZodObject<{
        key: z.ZodString;
        size: z.ZodNumber;
        order: z.ZodNumber;
        loc: z.ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>;
    }, z.core.$strip>>;
    visible: z.ZodDefault<z.ZodBoolean>;
    clearOverScan: z.ZodDefault<z.ZodUnion<readonly [z.ZodNumber, z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        width: z.ZodNumber;
        height: z.ZodNumber;
    }, z.core.$strip>, z.ZodObject<{
        signedWidth: z.ZodNumber;
        signedHeight: z.ZodNumber;
    }, z.core.$strip>, z.ZodObject<{
        clientX: z.ZodNumber;
        clientY: z.ZodNumber;
    }, z.core.$strip>]>>;
    loading: z.ZodDefault<z.ZodBoolean>;
}, z.core.$strip>;
export declare const linePlotMethodsZ: {
    getBounds: z.ZodFunction<z.ZodTuple<[], null>, z.ZodRecord<z.ZodString, z.ZodObject<{
        lower: z.ZodNumber;
        upper: z.ZodNumber;
    }, z.core.$strip>>>;
};
interface InternalState {
    instrumentation: Instrumentation;
    handleError: status.ErrorHandler;
    renderCtx: render.Context;
}
type Children = XAxis | tooltip.Tooltip | measure.Measure;
export declare class LinePlot extends aether.Composite<typeof linePlotStateZ, InternalState, Children, typeof linePlotMethodsZ> implements aether.HandlersFromSchema<typeof linePlotMethodsZ> {
    static readonly TYPE: string;
    schema: z.ZodObject<{
        container: z.ZodObject<{
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
        viewport: z.ZodObject<{
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
        hold: z.ZodDefault<z.ZodBoolean>;
        grid: z.ZodRecord<z.ZodString, z.ZodObject<{
            key: z.ZodString;
            size: z.ZodNumber;
            order: z.ZodNumber;
            loc: z.ZodEnum<{
                bottom: "bottom";
                left: "left";
                right: "right";
                top: "top";
            }>;
        }, z.core.$strip>>;
        visible: z.ZodDefault<z.ZodBoolean>;
        clearOverScan: z.ZodDefault<z.ZodUnion<readonly [z.ZodNumber, z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>, z.ZodTuple<[z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
            width: z.ZodNumber;
            height: z.ZodNumber;
        }, z.core.$strip>, z.ZodObject<{
            signedWidth: z.ZodNumber;
            signedHeight: z.ZodNumber;
        }, z.core.$strip>, z.ZodObject<{
            clientX: z.ZodNumber;
            clientY: z.ZodNumber;
        }, z.core.$strip>]>>;
        loading: z.ZodDefault<z.ZodBoolean>;
    }, z.core.$strip>;
    methods: {
        getBounds: z.ZodFunction<z.ZodTuple<[], null>, z.ZodRecord<z.ZodString, z.ZodObject<{
            lower: z.ZodNumber;
            upper: z.ZodNumber;
        }, z.core.$strip>>>;
    };
    afterUpdate(ctx: aether.Context): void;
    afterDelete(ctx: aether.Context): void;
    findByXDecimal(x: number): FindResult[];
    findByXValue(x: number): FindResult[];
    private get axes();
    private get tooltips();
    private get measures();
    private get exposure();
    private get loading();
    private renderAxes;
    private renderTooltips;
    getBounds(): AxesBounds;
    private renderMeasures;
    private calculatePlot;
    private render;
    requestRender(priority: render.Priority, reason: string): void;
}
export declare const REGISTRY: aether.ComponentRegistry;
export {};
//# sourceMappingURL=LinePlot.d.ts.map