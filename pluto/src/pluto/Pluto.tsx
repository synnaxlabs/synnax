// Copyright 2023 Synnax Labs, Inc.
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
import { Control } from "@/control";
import { Haul } from "@/haul";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { TelemProvider } from "@/telem/provider/Provider";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { Worker } from "@/worker";

// @ts-expect-error - unable to resolve valid vite import
// eslint-disable-next-line import/no-unresolved
import DefaultWorkerURL from "@/pluto/defaultWorker.ts&url";

import "@synnaxlabs/media/dist/style.css";

export interface ProviderProps
  extends PropsWithChildren,
    Partial<Theming.ProviderProps>,
    Synnax.ProviderProps {
  workerEnabled?: boolean;
  workerURL?: URL;
  alamos?: Alamos.ProviderProps;
  tooltip?: Tooltip.ConfigProps;
  triggers?: Triggers.ProviderProps;
  haul?: Haul.ProviderProps;
}

export const Provider = ({
  children,
  connParams,
  workerEnabled = true,
  workerURL,
  theme,
  toggleTheme,
  setTheme,
  tooltip,
  triggers,
  alamos,
  haul,
}: ProviderProps): ReactElement => {
  return (
    <Triggers.Provider {...triggers}>
      <Tooltip.Config {...tooltip}>
        <Haul.Provider {...haul}>
          <Worker.Provider url={workerURL ?? DefaultWorkerURL} enabled={workerEnabled}>
            <Aether.Provider workerKey="vis">
              <Alamos.Provider {...alamos}>
                <Status.Aggregator>
                  <Synnax.Provider connParams={connParams}>
                    <TelemProvider>
                      <Theming.Provider
                        theme={theme}
                        toggleTheme={toggleTheme}
                        setTheme={setTheme}
                      >
                        <Control.StateProvider>{children}</Control.StateProvider>
                      </Theming.Provider>
                    </TelemProvider>
                  </Synnax.Provider>
                </Status.Aggregator>
              </Alamos.Provider>
            </Aether.Provider>
          </Worker.Provider>
        </Haul.Provider>
      </Tooltip.Config>
    </Triggers.Provider>
  );
};
