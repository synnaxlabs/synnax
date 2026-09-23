import { Haul } from "@synnaxlabs/lyra/haul";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type CanDisabledProps } from "@synnaxlabs/lyra/util";
import { type PropsWithChildren, type ReactElement } from "react";
import { Alamos } from "../alamos";
import { Color } from "../color";
import { Synnax } from "../synnax";
import { Telem } from "../telem";
import { Theming } from "../theming";
import { Staleness } from "../vis/staleness";
export interface ProviderProps extends PropsWithChildren, Synnax.ProviderProps {
    theming?: Theming.ProviderProps;
    workerEnabled?: boolean;
    workerURL?: URL | string;
    alamos?: Alamos.ProviderProps;
    tooltip?: Tooltip.ConfigProps;
    triggers?: Triggers.ProviderProps;
    haul?: Haul.ProviderProps;
    telem?: CanDisabledProps<Telem.ProviderProps>;
    color?: Color.ProviderProps;
    staleness?: Staleness.ProviderProps;
}
export declare const Provider: ({ children, connParams, workerEnabled, workerURL, theming, tooltip, triggers, alamos, haul, telem, color, staleness, }: ProviderProps) => ReactElement;
//# sourceMappingURL=Pluto.d.ts.map