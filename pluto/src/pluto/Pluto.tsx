// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { type access } from "@/access/aether";
import { Aether } from "@/aether";
import { Alamos } from "@/alamos";
import { Arc } from "@/arc";
import { type Channel } from "@/channel";
import { Code } from "@/code";
import { Color } from "@/color";
import { type Device } from "@/device";
import { Flux } from "@/flux";
import { type Group } from "@/group";
import { Haul } from "@/haul";
import { type Label } from "@/label";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { type Ontology } from "@/ontology";
import { Panel } from "@/panel";
import DefaultWorkerURL from "@/pluto/defaultWorker.ts?url";
import { type Project } from "@/project";
import { type Rack } from "@/rack";
import { type Ranger } from "@/ranger";
import { type ranger } from "@/ranger/aether";
import { Schematic } from "@/schematic";
import { type Status } from "@/status";
import { Status as StatusBase } from "@/status/base";
import { Synnax } from "@/synnax";
import { Table } from "@/table";
import { type Task } from "@/task";
import { Telem } from "@/telem";
import { Control } from "@/telem/control";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { type User } from "@/user";
import { canDisable, type CanDisabledProps } from "@/util/canDisable";
import { type View } from "@/view";

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

export interface FluxStore extends Flux.Store {
  [ranger.FLUX_STORE_KEY]: ranger.FluxStore;
  [Label.FLUX_STORE_KEY]: Label.FluxStore;
  [Rack.FLUX_STORE_KEY]: Rack.FluxStore;
  [Device.FLUX_STORE_KEY]: Device.FluxStore;
  [Task.FLUX_STORE_KEY]: Task.FluxStore;
  [Project.FLUX_STORE_KEY]: Project.FluxStore;
  [Panel.FLUX_STORE_KEY]: Panel.FluxStore;
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
  [access.policy.FLUX_STORE_KEY]: access.policy.FluxStore;
  [access.role.FLUX_STORE_KEY]: access.role.FluxStore;
}

export const STORE_COMPOSERS: Flux.StoreComposers = {
  [Panel.FLUX_STORE_KEY]: Panel.STORE_COMPOSER,
  [LinePlot.FLUX_STORE_KEY]: LinePlot.STORE_COMPOSER,
  [Log.FLUX_STORE_KEY]: Log.STORE_COMPOSER,
  [Table.FLUX_STORE_KEY]: Table.STORE_COMPOSER,
  [Schematic.FLUX_STORE_KEY]: Schematic.STORE_COMPOSER,
  [Arc.FLUX_STORE_KEY]: Arc.STORE_COMPOSER,
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
        <CanDisableAether
          workerURL={workerURL ?? DefaultWorkerURL}
          workerEnabled={workerEnabled}
        >
          <Alamos.Provider {...alamos}>
            <StatusBase.Aggregator>
              <Synnax.Provider connParams={connParams}>
                <Flux.Provider composers={STORE_COMPOSERS}>
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
