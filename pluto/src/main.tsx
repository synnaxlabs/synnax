// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement } from "react";

import { Instrumentation } from "@synnaxlabs/alamos";

import { HaulProviderProps } from "./core/haul/HaulContext";
import DefaultWorkerURL from "./defaultWorker.ts?worker&url";

import { Client, ClientProviderProps } from "@/client";
import {
  Alamos,
  Aether,
  Haul,
  ThemeProviderProps,
  Theming,
  Triggers,
  Worker,
  Tooltip,
  TooltipConfigProps,
  TriggersProviderProps,
} from "@/core";
import { TelemProvider } from "@/telem/TelemProvider/TelemProvider";

import "@synnaxlabs/media/dist/style.css";

export interface PlutoProps
  extends PropsWithChildren,
    Partial<ThemeProviderProps>,
    ClientProviderProps {
  workerEnabled?: boolean;
  workerURL?: URL;
  instrumentation?: Instrumentation;
  tooltip?: TooltipConfigProps;
  triggers?: TriggersProviderProps;
  haul?: HaulProviderProps;
}

export const Pluto = ({
  children,
  connParams,
  workerEnabled = true,
  workerURL,
  theme,
  toggleTheme,
  setTheme,
  tooltip,
  instrumentation,
  triggers,
  haul,
}: PlutoProps): ReactElement => {
  return (
    <Alamos.Provider instrumentation={instrumentation}>
      <Triggers.Provider {...triggers}>
        <Tooltip.Config {...tooltip}>
          <Haul.Provider {...haul}>
            <Worker.Provider
              url={workerURL ?? DefaultWorkerURL}
              enabled={workerEnabled}
            >
              <Aether.Provider workerKey="vis">
                <Client.Provider connParams={connParams}>
                  <Theming.Provider
                    theme={theme}
                    toggleTheme={toggleTheme}
                    setTheme={setTheme}
                  >
                    <TelemProvider>{children}</TelemProvider>
                  </Theming.Provider>
                </Client.Provider>
              </Aether.Provider>
            </Worker.Provider>
          </Haul.Provider>
        </Tooltip.Config>
      </Triggers.Provider>
    </Alamos.Provider>
  );
};
