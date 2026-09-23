import { type xy } from "@synnaxlabs/x";
export interface Store {
    subscribe: (key: string, onChange: () => void) => () => void;
    get: (key: string) => xy.XY[];
    commit: (next: Map<string, xy.XY[]>) => void;
}
export declare const createStore: () => Store;
//# sourceMappingURL=store.d.ts.map