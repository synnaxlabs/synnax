// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@synnaxlabs/media/dist/style.css";

import { FC, type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { Alamos } from "@/alamos";
import { Channel } from "@/channel";
import { Haul } from "@/haul";
import DefaultWorkerURL from "@/pluto/defaultWorker.ts?url";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { Telem } from "@/telem";
import { Control } from "@/telem/control";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { Worker } from "@/worker";

type CanDisabledProps<T extends PropsWithChildren> = T & { disabled?: boolean };

const canDisable = <T extends PropsWithChildren>(C: FC<T>): FC<CanDisabledProps<T>> => {
  const O: FC<CanDisabledProps<T>> = ({ disabled = false, ...props }) =>
    disabled ? props.children : <C {...(props as T)} />;
  O.displayName = C.displayName;
  return O;
};

const CanDisableTelem = canDisable<Telem.ProviderProps>(Telem.Provider);
const CanDisableAether = canDisable<Aether.ProviderProps>(Aether.Provider);

export interface ProviderProps extends PropsWithChildren, Synnax.ProviderProps {
  theming?: Theming.ProviderProps;
  workerEnabled?: boolean;
  workerURL?: URL | string;
  alamos?: Alamos.ProviderProps;
  tooltip?: Tooltip.ConfigProps;
  triggers?: Triggers.ProviderProps;
  haul?: Haul.ProviderProps;
  channelAlias?: Channel.AliasProviderProps;
  telem?: CanDisabledProps<Telem.ProviderProps>;
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
  channelAlias,
  telem,
}: ProviderProps): ReactElement => (
  <Triggers.Provider {...triggers}>
    <Tooltip.Config {...tooltip}>
      <Haul.Provider {...haul}>
        <Worker.Provider url={workerURL ?? DefaultWorkerURL} enabled={workerEnabled}>
          <CanDisableAether workerKey="vis">
            <Alamos.Provider {...alamos}>
              <Status.Aggregator>
                <Synnax.Provider connParams={connParams}>
                  <Channel.AliasProvider {...channelAlias}>
                    <CanDisableTelem {...telem}>
                      <Theming.Provider {...theming}>
                        <Control.StateProvider>{children}</Control.StateProvider>
                      </Theming.Provider>
                    </CanDisableTelem>
                  </Channel.AliasProvider>
                </Synnax.Provider>
              </Status.Aggregator>
            </Alamos.Provider>
          </CanDisableAether>
        </Worker.Provider>
      </Haul.Provider>
    </Tooltip.Config>
  </Triggers.Provider>
);
