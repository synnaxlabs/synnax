import { channel } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

const PURPLE_HEX = color.construct("#635BFF");

export const Symbol = () => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        backgroundColor: color.cssString(color.setAlpha(PURPLE_HEX, 0.2)),
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        height: "8rem",
        padding: "0 1rem",
      }}
      align="center"
      justify="center"
    >
      <Icon.Task
        style={{
          width: "3.25rem",
          height: "3.25rem",
          color: color.cssString(PURPLE_HEX),
        }}
      />
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Configure Task
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="value" />
  </Align.Pack>
);
