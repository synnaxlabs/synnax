import { Form } from "@/arc/stage/read/Form";
import { type Config, Symbol } from "@/arc/stage/read/Read";
import { type Spec } from "@/arc/stage/types/spec";

export const SPEC: Spec<Config> = {
  key: "telem.read",
  name: "Read Channel",
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
