import { box, location, xy } from "@synnaxlabs/x";
export type Location = location.Outer | Partial<location.XY> | location.XY;
export interface Preference {
    targetCorner?: Location;
    dialogCorner?: Location;
}
export type LocationPreference = Location | Preference;
export interface Params {
    container: box.Crude;
    target: box.Crude;
    dialog: box.Crude;
    offset?: xy.Crude;
    initial?: LocationPreference;
    prefer?: LocationPreference | LocationPreference[];
    disable?: LocationPreference | LocationPreference[];
}
export declare const parseLocationOptions: (initial?: Location) => Partial<location.XY>;
export interface Return {
    targetCorner: location.XY;
    dialogCorner: location.XY;
    adjustedDialog: box.Box;
}
export declare const position: ({ container: containerCrude, target: targetCrude, dialog: dialogCrude, initial, prefer, disable, offset, }: Params) => Return;
//# sourceMappingURL=position.d.ts.map