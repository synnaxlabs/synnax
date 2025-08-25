import { Form } from "@/vis/slate/symbols/read/Form";
import { type Config, Symbol } from "@/vis/slate/symbols/read/Read";
import { type Spec } from "@/vis/slate/symbols/types/spec";

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
