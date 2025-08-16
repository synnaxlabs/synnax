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
import { Flux } from "@/flux";
import { Device } from "@/hardware/device";
import { Rack } from "@/hardware/rack";
import { Task } from "@/hardware/task";
import { Haul } from "@/haul";
import { Label } from "@/label";
import { Ontology } from "@/ontology";
import DefaultWorkerURL from "@/pluto/defaultWorker.ts?url";
import { Ranger } from "@/ranger";
import { ranger } from "@/ranger/aether";
import { Status } from "@/status";
import { Synnax } from "@/synnax";
import { Telem } from "@/telem";
import { Control } from "@/telem/control";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { canDisable, type CanDisabledProps } from "@/util/canDisable";
import { Worker } from "@/worker";
import { Workspace } from "@/workspace";

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
  telem?: CanDisabledProps<Telem.ProviderProps>;
  color?: Color.ProviderProps;
}

const STORE_CONFIG: Flux.StoreConfig<{
  ranges: ranger.FluxStore;
  labels: Label.FluxStore;
  racks: Rack.FluxStore;
  devices: Device.FluxStore;
  tasks: Task.FluxStore;
  workspaces: Workspace.FluxStore;
  relationships: Ontology.RelationshipFluxStore;
  rangeKV: Ranger.KVFluxStore;
  resources: Ontology.ResourceFluxStore;
  channels: Channel.FluxStore;
  rangeAliases: Ranger.AliasFluxStore;
}> = {
  ranges: ranger.STORE_CONFIG,
  labels: Label.STORE_CONFIG,
  racks: Rack.STORE_CONFIG,
  devices: Device.STORE_CONFIG,
  tasks: Task.STORE_CONFIG,
  workspaces: Workspace.STORE_CONFIG,
  relationships: Ontology.RELATIONSHIP_STORE_CONFIG,
  resources: Ontology.RESOURCE_STORE_CONFIG,
  rangeKV: Ranger.KV_STORE_CONFIG,
  channels: Channel.STORE_CONFIG,
  rangeAliases: Ranger.ALIAS_STORE_CONFIG,
};

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
        <Worker.Provider url={workerURL ?? DefaultWorkerURL} enabled={workerEnabled}>
          <CanDisableAether workerKey="vis">
            <Alamos.Provider {...alamos}>
              <Status.Aggregator>
                <Synnax.Provider connParams={connParams}>
                  <Flux.Provider storeConfig={STORE_CONFIG}>
                    <Color.Provider {...color}>
                      <Theming.Provider {...theming}>
                        <CanDisableTelem {...telem}>
                          <Control.StateProvider>{children}</Control.StateProvider>
                        </CanDisableTelem>
                      </Theming.Provider>
                    </Color.Provider>
                  </Flux.Provider>
                </Synnax.Provider>
              </Status.Aggregator>
            </Alamos.Provider>
          </CanDisableAether>
        </Worker.Provider>
      </Haul.Provider>
    </Tooltip.Config>
  </Triggers.Provider>
);
