import { type direction, xy } from "@synnaxlabs/x";
export interface SymbolPosition {
    position: xy.XY;
    direction: direction.Direction;
}
export declare const computeSymbolPositions: (points: xy.XY[], interval: number) => SymbolPosition[];
//# sourceMappingURL=computeSymbolPositions.d.ts.map