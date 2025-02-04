// Copyright 2025 Synnax Labs, Inc.
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
import { Channel } from "@/channel";
import { Color } from "@/color";
import { Haul } from "@/haul";
import DefaultWorkerURL from "@/pluto/defaultWorker.ts?url";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { Telem } from "@/telem";
import { Control } from "@/telem/control";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { canDisable, type CanDisabledProps } from "@/util/canDisable";
import { Worker } from "@/worker";

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
  channelAlias,
  telem,
  color,
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
                    <Color.Provider {...color}>
                      <Theming.Provider {...theming}>
                        <CanDisableTelem {...telem}>
                          <Control.StateProvider>{children}</Control.StateProvider>
                        </CanDisableTelem>
                      </Theming.Provider>
                    </Color.Provider>
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
