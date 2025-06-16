import { Form } from "@/vis/slate/symbols/systemlink/Form";
import { Update } from "@/vis/slate/symbols/systemlink/Update";
import { type types } from "@/vis/slate/symbols/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
  "systemlink.update": {
    key: "systemlink.update",
    name: "Update SystemLink Value",
    zIndex: 100,
    Form,
    Symbol: Update,
    Preview: Update,
    defaultProps: () => ({
      duration: 1000,
    }),
  },
};
