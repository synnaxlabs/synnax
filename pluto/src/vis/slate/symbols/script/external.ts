import { Form } from "@/vis/slate/symbols/script/Form";
import { type Config, Symbol } from "@/vis/slate/symbols/script/Script";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "script.run",
  name: "Run Script",
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
