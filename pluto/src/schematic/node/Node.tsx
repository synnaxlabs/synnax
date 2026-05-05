// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { useSyncedRef } from "@/hooks";
import { useKey } from "@/schematic/Context";
import { type Config, type Variant, resolveSpec } from "@/schematic/node/registry";
import { useDispatch, useSelectConfig } from "@/schematic/queries";
import { type Diagram } from "@/vis/diagram";

export const Node = ({
  nodeKey,
  position,
  selected,
  draggable,
}: Diagram.NodeProps): ReactElement | null => {
  const schematicKey = useKey();
  const config = useSelectConfig({ key: schematicKey, configKey: nodeKey });
  const configRef = useSyncedRef(config);
  const { update: dispatch } = useDispatch();
  const variant = (config as { variant?: Variant } | undefined)?.variant;
  const handleChange = useCallback(
    (next: record.Unknown) =>
      dispatch({
        key: schematicKey,
        actions: schematic.setConfig({
          key: nodeKey,
          config: { ...configRef.current, ...next },
        }),
      }),
    [nodeKey, schematicKey, dispatch],
  );
  if (config == null || variant == null) return null;
  const Spec = resolveSpec(variant);
  return (
    <Spec.Node
      nodeKey={nodeKey}
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
      config={config as Config}
      onConfigChange={handleChange}
      position={Spec.needsPosition === true ? position : undefined}
      selected={selected}
      draggable={draggable}
    />
  );
};
