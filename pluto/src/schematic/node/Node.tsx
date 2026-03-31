import { NotFoundError, schematic } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

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
  const { update: dispatch } = useDispatch();
  const variant = (nodeProps?.variant ?? null) as Symbol.Variant | null;

  const handleChange = useCallback(
    (props: object) => {
      if (variant == null) return;
      dispatch({
        key: schematicKey,
        actions: schematic.setProps({
          key: nodeKey,
          props: { variant, ...props },
        }),
      });
    },
    [nodeKey, schematicKey, variant, dispatch],
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
