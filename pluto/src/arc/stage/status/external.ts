import { type Config, Symbol } from "@/arc/stage/status/Change";
import { Form } from "@/arc/stage/status/Form";
import { type Spec } from "@/arc/stage/types/spec";

export const SPEC: Spec<Config> = {
  key: "set_status",
  name: "Change Status ",
  zIndex: 100,
  Form,
  Symbol,
  defaultProps: () => ({
    statusKey: "",
    variant: "success",
    message: "Notification",
  }),
  Preview: Symbol,
};

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
