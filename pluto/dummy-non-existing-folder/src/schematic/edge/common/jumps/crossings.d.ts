import { type xy } from "@synnaxlabs/x";
export interface Polyline {
    key: string;
    points: xy.XY[];
    order: number;
}
export declare const findCrossings: (polylines: Polyline[]) => Map<string, xy.XY[]>;
//# sourceMappingURL=crossings.d.ts.map