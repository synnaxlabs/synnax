import { Form } from "@/vis/slate/symbols/notification/Form";
import { type Config, Symbol } from "@/vis/slate/symbols/notification/Notification";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "notification",
  name: "Notification",
  zIndex: 100,
  Form,
  Symbol,
  defaultProps: () => ({
    variant: "success",
    message: "Notification",
  }),
  Preview: () => null,
};

export const REGISTRY = {
  [SPEC.key]: SPEC,
};
