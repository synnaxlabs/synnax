// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

import { Key } from "@/key";
import { type Config, resolveSpec } from "@/schematic/node/registry";
import { useDispatch, useSelectElementConfig } from "@/schematic/queries";
import { type Diagram } from "@/vis/diagram";

export const Renderer = ({
  position,
  ...rest
}: Diagram.NodeProps): ReactElement | null => {
  const { nodeKey } = rest;
  const key = Key.use<string>("Schematic.Node.Renderer");
  const config = useSelectElementConfig({ key, elKey: nodeKey });
  const { update: dispatch } = useDispatch();
  const handleChange = useCallback(
    (config: Partial<Config>) =>
      dispatch({ key, actions: schematic.setConfig({ key: nodeKey, config }) }),
    [nodeKey, key, dispatch],
  );
  // React flow can take time to unmount the node, meaning that we need to tolerate
  // temporarily undefinec configs.
  if (config == null) return null;
  const Spec = resolveSpec(config.variant);
  return (
    <Spec.Node
      config={config as Config}
      onConfigChange={handleChange}
      position={Spec.needsPosition === true ? position : undefined}
      {...rest}
    />
  );
};
