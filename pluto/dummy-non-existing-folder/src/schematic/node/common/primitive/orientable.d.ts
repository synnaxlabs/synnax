import { type color, type location } from "@synnaxlabs/x";
export interface OrientableProps {
    orientation?: location.Outer;
}
export interface SVGBasedProps extends OrientableProps {
    color?: color.Crude;
    scale?: number;
}
export declare const ZERO_PROPS: {
    readonly orientation: "left";
    readonly scale: 1;
};
//# sourceMappingURL=orientable.d.ts.map