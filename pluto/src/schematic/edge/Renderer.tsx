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
import { type Config } from "@/schematic/edge/config";
import { resolveSpec } from "@/schematic/edge/registry";
import { useDispatch, useSelectElementConfig } from "@/schematic/queries";
import { type Diagram } from "@/vis/diagram";

export const Renderer = (props: Diagram.EdgeProps): ReactElement | null => {
  const { edgeKey } = props;
  const key = Key.use<string>("Schematic.Edge.Renderer");
  const config = useSelectElementConfig({ key, elKey: edgeKey });
  const { update: dispatch } = useDispatch();
  const handleChange = useCallback(
    (config: Partial<Config>) =>
      dispatch({ key, actions: schematic.setConfig({ key: edgeKey, config }) }),
    [edgeKey, key, dispatch],
  );
  // React flow can take time to unmount the edge, meaning that we need to tolerate
  // temporarily undefinec configs.
  if (config == null) return null;
  const Spec = resolveSpec(config.variant);
  return <Spec.Edge config={config as Config} onChange={handleChange} {...props} />;
};
