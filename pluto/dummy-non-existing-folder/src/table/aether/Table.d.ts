import { box, scale } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../aether/aether";
import { status } from "../../status/aether";
import { render } from "../../vis/render";
export declare const tableStateZ: z.ZodObject<{
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
    scroll: z.ZodDefault<z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>>;
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
    visible: z.ZodDefault<z.ZodBoolean>;
    autoRenderInterval: z.ZodDefault<z.ZodNumber>;
}, z.core.$strip>;
interface CellProps {
    viewportScale: scale.XY;
}
export interface Cell extends aether.Component {
    /** The cell's position in the table's unscrolled layout coordinates. */
    readonly box: box.Box;
    render: ({ viewportScale }: CellProps) => void;
}
interface InternalState {
    renderCtx: render.Context;
    handleError: status.ErrorHandler;
    autoRenderInterval: ReturnType<typeof setInterval>;
}
export declare class Table extends aether.Composite<typeof tableStateZ, InternalState, Cell> {
    static readonly TYPE = "Table";
    static readonly stateZ: z.ZodObject<{
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
        scroll: z.ZodDefault<z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>>;
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
        visible: z.ZodDefault<z.ZodBoolean>;
        autoRenderInterval: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
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
        scroll: z.ZodDefault<z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>>;
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
        visible: z.ZodDefault<z.ZodBoolean>;
        autoRenderInterval: z.ZodDefault<z.ZodNumber>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    render(): render.Cleanup | undefined;
    private requestRender;
}
export declare const REGISTRY: {
    Table: typeof Table;
};
export {};
//# sourceMappingURL=Table.d.ts.map