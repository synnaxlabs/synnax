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
import { Arc } from "@/arc";
import { Channel } from "@/channel";
import { Color } from "@/color";
import { Device } from "@/device";
import { Flux } from "@/flux";
import { Group } from "@/group";
import { Haul } from "@/haul";
import { Label } from "@/label";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import DefaultWorkerURL from "@/pluto/defaultWorker.ts?url";
import { Rack } from "@/rack";
import { Ranger } from "@/ranger";
import { ranger } from "@/ranger/aether";
import { Schematic } from "@/schematic";
import { Status } from "@/status";
import { Status as StatusCore } from "@/status/core";
import { Synnax } from "@/synnax";
import { Table } from "@/table";
import { Task } from "@/task";
import { Telem } from "@/telem";
import { Control } from "@/telem/control";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { User } from "@/user";
import { canDisable, type CanDisabledProps } from "@/util/canDisable";
import { View } from "@/view";
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

export const FLUX_STORE_CONFIG: Flux.StoreConfig<{
  [ranger.FLUX_STORE_KEY]: ranger.FluxStore;
  [Label.FLUX_STORE_KEY]: Label.FluxStore;
  [Rack.FLUX_STORE_KEY]: Rack.FluxStore;
  [Device.FLUX_STORE_KEY]: Device.FluxStore;
  [Task.FLUX_STORE_KEY]: Task.FluxStore;
  [Workspace.FLUX_STORE_KEY]: Workspace.FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ranger.RANGE_KV_FLUX_STORE_KEY]: Ranger.KVFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [Channel.FLUX_STORE_KEY]: Channel.FluxStore;
  [Ranger.RANGE_ALIASES_FLUX_STORE_KEY]: Ranger.AliasFluxStore;
  [Schematic.Symbol.FLUX_STORE_KEY]: Schematic.Symbol.FluxStore;
  [Group.FLUX_STORE_KEY]: Group.FluxStore;
  [Status.FLUX_STORE_KEY]: Status.FluxStore;
  [Arc.FLUX_STORE_KEY]: Arc.FluxStore;
  [LinePlot.FLUX_STORE_KEY]: LinePlot.FluxStore;
  [Log.FLUX_STORE_KEY]: Log.FluxStore;
  [Table.FLUX_STORE_KEY]: Table.FluxStore;
  [Schematic.FLUX_STORE_KEY]: Schematic.FluxStore;
  [User.FLUX_STORE_KEY]: User.FluxStore;
  [View.FLUX_STORE_KEY]: View.FluxStore;
}> = {
  [ranger.FLUX_STORE_KEY]: ranger.FLUX_STORE_CONFIG,
  [Label.FLUX_STORE_KEY]: Label.FLUX_STORE_CONFIG,
  [Rack.FLUX_STORE_KEY]: Rack.FLUX_STORE_CONFIG,
  [Device.FLUX_STORE_KEY]: Device.FLUX_STORE_CONFIG,
  [Task.FLUX_STORE_KEY]: Task.FLUX_STORE_CONFIG,
  [Workspace.FLUX_STORE_KEY]: Workspace.FLUX_STORE_CONFIG,
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RELATIONSHIP_FLUX_STORE_CONFIG,
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.RESOURCE_FLUX_STORE_CONFIG,
  [Ranger.RANGE_KV_FLUX_STORE_KEY]: Ranger.KV_FLUX_STORE_CONFIG,
  [Channel.FLUX_STORE_KEY]: Channel.FLUX_STORE_CONFIG,
  [Ranger.RANGE_ALIASES_FLUX_STORE_KEY]: Ranger.ALIAS_FLUX_STORE_CONFIG,
  [Schematic.Symbol.FLUX_STORE_KEY]: Schematic.Symbol.STORE_CONFIG,
  [Group.FLUX_STORE_KEY]: Group.FLUX_STORE_CONFIG,
  [Status.FLUX_STORE_KEY]: Status.FLUX_STORE_CONFIG,
  [LinePlot.FLUX_STORE_KEY]: LinePlot.FLUX_STORE_CONFIG,
  [Log.FLUX_STORE_KEY]: Log.FLUX_STORE_CONFIG,
  [Table.FLUX_STORE_KEY]: Table.FLUX_STORE_CONFIG,
  [Schematic.FLUX_STORE_KEY]: Schematic.FLUX_STORE_CONFIG,
  [User.FLUX_STORE_KEY]: User.FLUX_STORE_CONFIG,
  [Arc.FLUX_STORE_KEY]: Arc.FLUX_STORE_CONFIG,
  [View.FLUX_STORE_KEY]: View.FLUX_STORE_CONFIG,
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
              <StatusCore.Aggregator>
                <Synnax.Provider connParams={connParams}>
                  <Flux.Provider storeConfig={FLUX_STORE_CONFIG}>
                    <Color.Provider {...color}>
                      <Theming.Provider {...theming}>
                        <CanDisableTelem {...telem}>
                          <Control.StateProvider>{children}</Control.StateProvider>
                        </CanDisableTelem>
                      </Theming.Provider>
                    </Color.Provider>
                  </Flux.Provider>
                </Synnax.Provider>
              </StatusCore.Aggregator>
            </Alamos.Provider>
          </CanDisableAether>
        </Worker.Provider>
      </Haul.Provider>
    </Tooltip.Config>
  </Triggers.Provider>
);
