import { Icon } from "@synnaxlabs/media";
import { color, status } from "@synnaxlabs/x";
import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const config = z.object({
  variant: status.variantZ,
  message: z.string(),
  description: z.string().optional(),
});

export interface Config extends z.infer<typeof config> {}

const RED_HEX = color.construct("#DC136C");

export const Symbol = ({ message }: Config) => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        height: "8rem",
        width: "4rem",
        backgroundColor: color.cssString(color.setAlpha(RED_HEX, 0.2)),
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
      }}
      align="center"
      justify="center"
    >
      <Icon.Notification
        style={{
          width: "2.5rem",
          height: "2.5rem",
          color: color.cssString(RED_HEX),
        }}
      />
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" size="tiny">
      <Text.Text level="small" weight={500} shade={9}>
        Change Status
      </Text.Text>
      <Text.Text level="p" weight={500} shade={11}>
        {message}
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="value" />
  </Align.Pack>
);
