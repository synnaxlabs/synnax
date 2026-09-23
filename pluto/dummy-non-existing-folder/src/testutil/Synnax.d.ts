import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren } from "react";
import { type aether } from "../aether/aether";
import { telem } from "../telem/aether";
import { canvasTest } from "../vis/render/test";
export interface CreateSynnaxWrapperParams {
    client: Client | null;
    /** Extra aether components merged into the test render registry. */
    additionalRegistry?: aether.ComponentRegistry;
    /**
     * Seeds the given fake render context into the aether tree, so canvas-rendered
     * components can mount and record draw calls without a real canvas. Construct one
     * with {@link canvasTest.record} and keep the reference for assertions.
     */
    renderContext?: canvasTest.Recorder;
    /**
     * When defined, mounts the telemetry provider stack so components that resolve
     * telem sources (e.g. value cells) can mount. The factories are added to the
     * production factory set; pass `[new telemTest.TestFactory()]` to resolve
     * telemTest source and sink specs.
     */
    telemFactories?: telem.Factory[];
    /** Connection status the Synnax context reports; defaults to disconnected. */
    connectionStatus?: connection.Status;
}
export declare const createSynnaxWrapper: ({ client, additionalRegistry, renderContext, telemFactories, connectionStatus, }: CreateSynnaxWrapperParams) => FC<PropsWithChildren>;
export declare const createAsyncSynnaxWrapper: (params: CreateSynnaxWrapperParams) => Promise<FC<PropsWithChildren>>;
//# sourceMappingURL=Synnax.d.ts.map