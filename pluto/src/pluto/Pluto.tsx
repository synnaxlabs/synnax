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
import { annotation as aetherAnnotation } from "@/annotation/aether";
import { Channel } from "@/channel";
import { Color } from "@/color";
import { Effect } from "@/effect";
import { Flux } from "@/flux";
import { Group } from "@/group";
import { Device } from "@/hardware/device";
import { Rack } from "@/hardware/rack";
import { Task } from "@/hardware/task";
import { Haul } from "@/haul";
import { Label } from "@/label";
import { ontology as aetherOntology } from "@/ontology/aether";
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

export const FLUX_STORE_CONFIG: Flux.StoreConfig<{
  [ranger.FLUX_STORE_KEY]: ranger.FluxStore;
  [aetherAnnotation.FLUX_STORE_KEY]: aetherAnnotation.FluxStore;
  [Label.FLUX_STORE_KEY]: Label.FluxStore;
  [Rack.FLUX_STORE_KEY]: Rack.FluxStore;
  [Device.FLUX_STORE_KEY]: Device.FluxStore;
  [Task.FLUX_STORE_KEY]: Task.FluxStore;
  [Workspace.FLUX_STORE_KEY]: Workspace.FluxStore;
  [aetherOntology.RELATIONSHIPS_FLUX_STORE_KEY]: aetherOntology.RelationshipFluxStore;
  [Ranger.RANGE_KV_FLUX_STORE_KEY]: Ranger.KVFluxStore;
  [aetherOntology.RESOURCES_FLUX_STORE_KEY]: aetherOntology.ResourceFluxStore;
  [Channel.FLUX_STORE_KEY]: Channel.FluxStore;
  [Ranger.RANGE_ALIASES_FLUX_STORE_KEY]: Ranger.AliasFluxStore;
  [Group.FLUX_STORE_KEY]: Group.FluxStore;
  [Effect.FLUX_STORE_KEY]: Effect.FluxStore;
}> = {
  [ranger.FLUX_STORE_KEY]: ranger.FLUX_STORE_CONFIG,
  [aetherAnnotation.FLUX_STORE_KEY]: aetherAnnotation.STORE_CONFIG,
  [Label.FLUX_STORE_KEY]: Label.FLUX_STORE_CONFIG,
  [Rack.FLUX_STORE_KEY]: Rack.FLUX_STORE_CONFIG,
  [Device.FLUX_STORE_KEY]: Device.FLUX_STORE_CONFIG,
  [Task.FLUX_STORE_KEY]: Task.FLUX_STORE_CONFIG,
  [Workspace.FLUX_STORE_KEY]: Workspace.FLUX_STORE_CONFIG,
  [aetherOntology.RELATIONSHIPS_FLUX_STORE_KEY]:
    aetherOntology.RELATIONSHIP_STORE_CONFIG,
  [aetherOntology.RESOURCES_FLUX_STORE_KEY]: aetherOntology.RESOURCE_STORE_CONFIG,
  [Ranger.RANGE_KV_FLUX_STORE_KEY]: Ranger.KV_STORE_CONFIG,
  [Channel.FLUX_STORE_KEY]: Channel.STORE_CONFIG,
  [Ranger.RANGE_ALIASES_FLUX_STORE_KEY]: Ranger.ALIAS_STORE_CONFIG,
  [Effect.FLUX_STORE_KEY]: Effect.STORE_CONFIG,
  [Group.FLUX_STORE_KEY]: Group.FLUX_STORE_CONFIG,
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
              </Status.Aggregator>
            </Alamos.Provider>
          </CanDisableAether>
        </Worker.Provider>
      </Haul.Provider>
    </Tooltip.Config>
  </Triggers.Provider>
);
