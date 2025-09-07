import { Form } from "@/arc/symbols/stable/Form";
import { StableFor } from "@/arc/symbols/stable/StableFor";
import { type types } from "@/arc/symbols/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
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
