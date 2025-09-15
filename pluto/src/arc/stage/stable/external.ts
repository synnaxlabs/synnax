import { TimeSpan } from "@synnaxlabs/x";

import { Form } from "@/arc/stage/stable/Form";
import { StableFor } from "@/arc/stage/stable/StableFor";
import { type types } from "@/arc/stage/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
  stable_for: {
    key: "stable_for",
    name: "Stable For",
    zIndex: 100,
    Form,
    Symbol: StableFor,
    Preview: StableFor,
    defaultProps: () => ({
      duration: TimeSpan.milliseconds(250).valueOf(),
    }),
  },
};
