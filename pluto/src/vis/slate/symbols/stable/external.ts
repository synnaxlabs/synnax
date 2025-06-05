import { Form } from "@/vis/slate/symbols/stable/Form";
import { StableFor } from "@/vis/slate/symbols/stable/StableFor";
import { type types } from "@/vis/slate/symbols/types";

export const REGISTRY: Record<string, types.Spec<any>> = {
  stable_for: {
    key: "stable_for",
    name: "Stable For",
    zIndex: 100,
    Form,
    Symbol: StableFor,
    Preview: StableFor,
    defaultProps: () => ({
      duration: 1000,
    }),
  },
};
