import { type CSSProperties } from "react";
interface CSSGridEntry {
    startLabel: string;
    endLabel: string;
    size: number | string;
}
export declare class CSSGridBuilder {
    rows: CSSGridEntry[];
    columns: CSSGridEntry[];
    prefix: string;
    constructor(prefix?: string);
    row(start: string, end: string, size: number | string): this;
    col(start: string, end: string, size: number | string): this;
    build(): CSSProperties;
}
export {};
//# sourceMappingURL=grid.d.ts.map