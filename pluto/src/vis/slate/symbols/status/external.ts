import { type Config, Symbol } from "@/vis/slate/symbols/status/Change";
import { Form } from "@/vis/slate/symbols/status/Form";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "status.change",
  name: "Change Status ",
  zIndex: 100,
  Form,
  Symbol,
  defaultProps: () => ({
    variant: "success",
    message: "Notification",
  }),
  Preview: Symbol,
};

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
