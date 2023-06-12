import { PropsWithChildren, ReactElement } from "react";

import { ClientProviderProps } from "./client/Context";

import { Client } from "@/client";
import { Aether, Haul, ThemeProviderProps, Theming, Triggers } from "@/core";
import { Worker } from "@/core/worker";
import { TelemProvider } from "@/telem/Context";

export interface PlutoProps
  extends PropsWithChildren,
    Partial<ThemeProviderProps>,
    ClientProviderProps {
  workerEnabled?: boolean;
  workerURL?: URL;
}

const WORKER_URL = new URL("./plutoWorker.ts", import.meta.url);

export const Pluto = ({
  children,
  connParams,
  workerEnabled = true,
  workerURL = WORKER_URL,
  theme,
  toggleTheme,
  setTheme,
}: PlutoProps): ReactElement => (
  <Theming.Provider theme={theme} toggleTheme={toggleTheme} setTheme={setTheme}>
    <Triggers.Provider>
      <Haul.Provider>
        <Worker.Provider url={workerURL} enabled={workerEnabled}>
          <Aether.Provider workerKey="vis">
            <Client.Provider connParams={connParams}>
              <TelemProvider>{children}</TelemProvider>
            </Client.Provider>
          </Aether.Provider>
        </Worker.Provider>
      </Haul.Provider>
    </Triggers.Provider>
  </Theming.Provider>
);
