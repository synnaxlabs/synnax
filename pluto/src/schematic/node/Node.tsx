// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, schematic } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { useSyncedRef } from "@/hooks";
import { useKey } from "@/schematic/Context";
import { useDispatch, useSelectProps } from "@/schematic/queries";
import { Symbol } from "@/schematic/symbol";
import { type Diagram } from "@/vis/diagram";

export const Node = ({
  nodeKey,
  position,
  selected,
  draggable,
}: Diagram.NodeProps): ReactElement | null => {
  const schematicKey = useKey();
  const nodeProps = useSelectProps({ key: schematicKey, propKey: nodeKey });
  const propsRef = useSyncedRef(nodeProps);
  const { update: dispatch } = useDispatch();
  const variant = nodeProps?.variant as Symbol.Variant | undefined;
  const handleChange = useCallback(
    (props: record.Unknown) =>
      dispatch({
        key: schematicKey,
        actions: schematic.setProps({
          key: nodeKey,
          props: { ...propsRef.current, ...props },
        }),
      }),
    [nodeKey, schematicKey, dispatch],
  );
  if (nodeProps == null || variant == null) return null;
  const Spec = Symbol.REGISTRY[variant];
  if (Spec == null) throw new NotFoundError(`Symbol ${variant} not found`);
  return (
    <Spec.Symbol
      key={nodeKey}
      nodeKey={nodeKey}
      position={position}
      selected={selected}
      draggable={draggable}
      onChange={handleChange}
      data={nodeProps}
    />
  );
};
