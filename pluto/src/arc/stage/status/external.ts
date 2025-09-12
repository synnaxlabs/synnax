import { uuid } from "@synnaxlabs/x";

import { type Config, Symbol } from "@/arc/stage/status/Change";
import { Form } from "@/arc/stage/status/Form";
import { type Spec } from "@/arc/stage/types/spec";

export const SPEC: Spec<Config> = {
  key: "status.change",
  name: "Change Status ",
  zIndex: 100,
  Form,
  Symbol,
  defaultProps: () => ({
    key: uuid.create(),
    variant: "success",
    message: "Notification",
  }),
  Preview: Symbol,
};

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
