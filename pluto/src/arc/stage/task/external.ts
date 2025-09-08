import { type Config, Symbol } from "@/arc/stage/task/Configure";
import { Form } from "@/arc/stage/task/Form";
import { type Spec } from "@/arc/stage/types/spec";

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
