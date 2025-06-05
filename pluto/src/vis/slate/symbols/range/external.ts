import { color } from "@synnaxlabs/x";

import { Create, type CreateConfig } from "@/vis/slate/symbols/range/Create";
import { CreateForm } from "@/vis/slate/symbols/range/CreateForm";
import { type types } from "@/vis/slate/symbols/types";

const CREATE_RANGE: types.Spec<CreateConfig> = {
  key: "range.create",
  name: "Create Range",
  zIndex: 100,
  Form: CreateForm,
  Symbol: Create,
  Preview: Create,
  defaultProps: () => ({
    range: {
      name: "New Range",
      color: color.hex(color.construct("#000000")),
    },
  }),
};

export const REGISTRY: Record<string, types.Spec<any>> = {
  [CREATE_RANGE.key]: CREATE_RANGE,
};
