import { type status } from "../../status/aether";
import { type telem } from "./";
import { type Client } from "./remote";
export interface CreateOptions {
    onStatusChange?: status.Adder;
}
export interface Factory {
    type: string;
    create: (spec: telem.Spec, options?: CreateOptions) => telem.Telem | null;
}
export declare class CompoundFactory {
    factories: Factory[];
    type: string;
    constructor(factories: Factory[]);
    add(factory: Factory): void;
    create(props: telem.Spec, options?: CreateOptions): telem.Telem | null;
}
export declare const createFactory: (client?: Client | null, extra?: Factory[]) => CompoundFactory;
//# sourceMappingURL=factory.d.ts.map