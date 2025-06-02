import { Icon } from "@synnaxlabs/media";
import z from "zod";

import { Align } from "@/align";
import { Icon as PIcon } from "@/icon";
import { status } from "@/status/aether";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const config = z.object({
  variant: status.variantZ,
  message: z.string(),
  description: z.string().optional(),
});

export interface Config extends z.infer<typeof config> {}

export const Symbol = () => (
  <Align.Pack x align="center" background={1} bordered borderShade={5} rounded={1}>
    <PIcon.Icon
      style={{
        padding: "0.5rem",
        paddingBottom: "0.25rem",
        background: "#DC136C",
        borderTopLeftRadius: "0.5rem",
        borderBottomLeftRadius: "0.5rem",
      }}
    >
      <Icon.Notification
        style={{
          width: "3.25rem",
          height: "3.25rem",
          color: "var(--pluto-gray-l0)",
        }}
      />
    </PIcon.Icon>
    <Text.Text level="p" weight={500} style={{ padding: "0 2rem" }}>
      Send Notification
    </Text.Text>
    <Handle.Sink location="left" id="input" />
  </Align.Pack>
);
