import { type panel } from "@synnaxlabs/client";

import { context } from "@/context";

export const [TabKeyContext, useTabKey] = context.create<panel.TabKey>({
  displayName: "Panel.TabKeyContext",
  providerName: "Panel.TabKeyContext",
});

export const [KeyContext, useKey] = context.create<panel.Key>({
  displayName: "Panel.KeyContext",
  providerName: "Panel.KeyContext",
});
