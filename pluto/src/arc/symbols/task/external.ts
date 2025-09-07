import { type Config, Symbol } from "@/arc/symbols/task/Configure";
import { Form } from "@/arc/symbols/task/Form";
import { type Spec } from "@/arc/symbols/types/spec";

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

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
