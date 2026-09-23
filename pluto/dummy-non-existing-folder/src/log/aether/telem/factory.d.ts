import { type telem } from "../../../telem/aether";
export declare class LogFactory implements telem.Factory {
    type: string;
    private readonly client;
    constructor(client: telem.Client | null);
    create(spec: telem.Spec, options?: telem.CreateOptions): telem.Telem | null;
}
//# sourceMappingURL=factory.d.ts.map