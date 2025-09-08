import { Form } from "@/arc/stage/source/Form";
import { type Config, Symbol } from "@/arc/stage/source/Source";
import { type Spec } from "@/arc/stage/types/spec";

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
