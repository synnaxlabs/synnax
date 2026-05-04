import { type record } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { REGISTRY, type Variant } from "@/schematic/edge/registry";
import { Select } from "@/select";

const SELECT_DATA: record.KeyedNamed<Variant>[] = Object.values(REGISTRY).map(
  ({ key, name }) => ({ key, name }),
);

const SELECT_STYLE: CSSProperties = { width: "25rem" };

export interface SelectVariantProps extends Omit<
  Select.StaticProps<Variant>,
  "data" | "resourceName"
> {}

export const SelectVariant = (props: SelectVariantProps): ReactElement => (
  <Select.Static
    {...props}
    data={SELECT_DATA}
    resourceName="edge type"
    style={{ ...SELECT_STYLE, ...props.style }}
  />
);
