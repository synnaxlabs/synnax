import { type Synnax as SynnaxClient } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren } from "react";
import { type aether } from "../../aether/aether";
import { telem } from "../aether";
export interface CreateTestWrapperOptions {
    registry: aether.ComponentRegistry;
    client?: SynnaxClient | null;
    telemFactories?: telem.Factory[];
}
export declare const createTestWrapper: (options: CreateTestWrapperOptions) => FC<PropsWithChildren>;
//# sourceMappingURL=wrapper.d.ts.map