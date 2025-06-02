import { type Config, Symbol } from "@/vis/slate/symbols/annotation/Create";
import { Form } from "@/vis/slate/symbols/annotation/Form";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "annotation.create",
  name: "Create",
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
