import { Form } from "@/vis/slate/symbols/source/Form";
import { type Config, Symbol } from "@/vis/slate/symbols/source/Source";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "source",
  name: "Source",
  Form,
  Symbol,
  defaultProps: () => ({
    channels: [],
  }),
  Preview: Symbol,
  zIndex: 0,
};

export const REGISTRY = {
  [SPEC.key]: SPEC,
};
