import { Form } from "@/vis/slate/symbols/source/Form";
import { type Config, Symbol } from "@/vis/slate/symbols/source/Source";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "telem.source",
  name: "Telemetry Source",
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
