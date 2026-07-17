// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { Alamos } from "@/alamos";
import { Arc } from "@/arc";
import { Code } from "@/code";
import { Color } from "@/color";
import { Flux } from "@/flux";
import { Haul } from "@/haul";
import DefaultWorkerURL from "@/pluto/defaultWorker.ts?url";
import { Status as StatusBase } from "@/status/base";
import { Synnax } from "@/synnax";
import { Telem } from "@/telem";
import { Control } from "@/telem/control";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { canDisable, type CanDisabledProps } from "@/util/canDisable";

const CanDisableTelem = canDisable<Telem.ProviderProps>(Telem.Provider);
const CanDisableAether = canDisable<Aether.ProviderProps>(Aether.Provider);

const ARC_LANGUAGES = [Arc.LANGUAGE];

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
}

export const Provider = ({
  children,
  connParams,
  workerEnabled = true,
  workerURL,
  theming,
  tooltip,
  triggers,
  alamos,
  haul,
  telem,
  color,
}: ProviderProps): ReactElement => (
  <Triggers.Provider {...triggers}>
    <Tooltip.Config {...tooltip}>
      <Haul.Provider {...haul}>
        <CanDisableAether
          workerURL={workerURL ?? DefaultWorkerURL}
          workerEnabled={workerEnabled}
        >
          <Alamos.Provider {...alamos}>
            <StatusBase.Aggregator>
              <Synnax.Provider connParams={connParams}>
                <Flux.Provider>
                  <Color.Provider {...color}>
                    <Theming.Provider {...theming}>
                      <Code.Provider languages={ARC_LANGUAGES}>
                        <CanDisableTelem {...telem}>
                          <Control.StateProvider>{children}</Control.StateProvider>
                        </CanDisableTelem>
                      </Code.Provider>
                    </Theming.Provider>
                  </Color.Provider>
                </Flux.Provider>
              </Synnax.Provider>
            </StatusBase.Aggregator>
          </Alamos.Provider>
        </CanDisableAether>
      </Haul.Provider>
    </Tooltip.Config>
  </Triggers.Provider>
);
