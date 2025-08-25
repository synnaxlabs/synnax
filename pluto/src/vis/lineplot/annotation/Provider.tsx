// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";

import { Aether } from "@/aether";
import { type RenderProp } from "@/component/renderProp";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { annotation } from "@/vis/lineplot/annotation/aether";

export interface ProviderProps extends Aether.ComponentProps {
  parents: ontology.ID[];
  menu?: RenderProp<annotation.SelectedState>;
}

export const Provider = ({
  aetherKey,
  parents,
  menu,
  ...rest
}: ProviderProps): null => {
  const cKey = useUniqueKey(aetherKey);
  Aether.use({
    aetherKey: cKey,
    type: annotation.Provider.TYPE,
    schema: annotation.providerStateZ,
    initialState: { ...rest, cursor: null, hovered: null, count: 0, parents },
  });
  return null;
};
