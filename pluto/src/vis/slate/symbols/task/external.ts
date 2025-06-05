import { type Config, Symbol } from "@/vis/slate/symbols/task/Configure";
import { Form } from "@/vis/slate/symbols/task/Form";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "task.configure",
  name: "Configure Task",
  Form,
  Symbol,
  defaultProps: () => ({
    channel: 0,
  }),
  Preview: Symbol,
  zIndex: 0,
};

export const REGISTRY = {
  [SPEC.key]: SPEC,
};
