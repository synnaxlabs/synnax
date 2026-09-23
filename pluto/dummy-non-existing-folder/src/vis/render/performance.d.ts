import { type destructor, TimeSpan } from "@synnaxlabs/x";
export declare class Tracker {
    private readonly entries;
    private readonly target;
    constructor(target: TimeSpan);
    measure(key: string): [number, destructor.Destructor];
    updateLevels(): void;
    levels(): Map<string, number>;
}
//# sourceMappingURL=performance.d.ts.map