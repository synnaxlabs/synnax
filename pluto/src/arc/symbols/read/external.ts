import { Form } from "@/arc/symbols/read/Form";
import { type Config, Symbol } from "@/arc/symbols/read/Read";
import { type Spec } from "@/arc/symbols/types/spec";

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
