import { scale } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
import { status } from "../../../status/aether";
import { render } from "../../render";
export declare const diagramStateZ: z.ZodObject<{
    position: z.ZodObject<{
        x: z.ZodNumber;
        y: z.ZodNumber;
    }, z.core.$strip>;
    zoom: z.ZodNumber;
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
    autoRenderInterval: z.ZodOptional<z.ZodNumber>;
}, z.core.$strip>;
interface ElementProps {
    viewportScale?: scale.XY;
}
export interface Element extends aether.Component {
    render?: (props: ElementProps) => void;
}
interface InternalState {
    renderCtx: render.Context;
    viewportScale: scale.XY;
    handleError: status.ErrorHandler;
    autoRenderInterval: ReturnType<typeof setInterval>;
}
export declare class Diagram extends aether.Composite<typeof diagramStateZ, InternalState, Element> {
    static readonly TYPE = "Diagram";
    static readonly stateZ: z.ZodObject<{
        position: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        zoom: z.ZodNumber;
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
        autoRenderInterval: z.ZodOptional<z.ZodNumber>;
    }, z.core.$strip>;
    schema: z.ZodObject<{
        position: z.ZodObject<{
            x: z.ZodNumber;
            y: z.ZodNumber;
        }, z.core.$strip>;
        zoom: z.ZodNumber;
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
        autoRenderInterval: z.ZodOptional<z.ZodNumber>;
    }, z.core.$strip>;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    render(): render.Cleanup | undefined;
    private requestRender;
}
export declare const REGISTRY: {
    Diagram: typeof Diagram;
};
export {};
//# sourceMappingURL=Diagram.d.ts.map