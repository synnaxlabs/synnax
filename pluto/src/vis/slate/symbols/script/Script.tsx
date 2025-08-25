import { channel } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import z from "zod/v4";

import { Flex } from "@/flex";
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
  <Flex.Box pack x align="center" background={1} bordered borderColor={6} rounded={1}>
    <Flex.Box
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
      <Icon.TypeScript
        style={{
          width: "3.25rem",
          height: "3.25rem",
          color: color.cssString(PURPLE_HEX),
        }}
      />
    </Flex.Box>
    <Divider.Divider y color={5} />
    <Flex.Box style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} color={9}>
        Run Script
      </Text.Text>
    </Flex.Box>
    <Handle.Sink location="left" id="value" />
    <Handle.Source location="right" id="output" />
  </Flex.Box>
);
