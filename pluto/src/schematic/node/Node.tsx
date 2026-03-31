import { schematic } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

import { useKey } from "@/schematic/Context";
import { useDispatch, useRetrieve } from "@/schematic/queries";
import { Symbol } from "@/schematic/symbol";
import { type Diagram } from "@/vis/diagram";

export const Node = ({
  nodeKey,
  position,
  selected,
}: Diagram.NodeProps): ReactElement | null => {
  const key = useKey();
  const { data: doc } = useRetrieve({ key });
  const { update: dispatch } = useDispatch();
  const nodeProps = doc?.props?.[nodeKey] as Record<string, unknown> | undefined;
  const variant = (nodeProps?.key ?? null) as Symbol.Variant | null;

  const handleChange = useCallback(
    (props: object) => {
      if (variant == null) return;
      dispatch({
        key,
        actions: schematic.setProps({
          key: nodeKey,
          props: { key: variant, ...props },
        }),
      });
    },
    [nodeKey, key, variant, dispatch],
  );

  if (nodeProps == null || variant == null) return null;

  const C = Symbol.REGISTRY[variant];
  if (C == null) throw new Error(`Symbol ${variant} not found`);

  const { key: _, ...rest } = nodeProps;

  return (
    <C.Symbol
      key={variant}
      id={nodeKey}
      symbolKey={nodeKey}
      position={position}
      selected={selected}
      onChange={handleChange}
      {...rest}
    />
  );
};
