import { color } from "@synnaxlabs/x";
import z from "zod/v4";

import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

const PURPLE_HEX = color.construct("#635BFF");

export const Count = () => (
  <Flex.Box pack x align="center" background={1} bordered borderColor={6} rounded={1}>
    <Flex.Box
      style={{
        height: "8rem",
        backgroundColor: color.cssString(color.setAlpha(PURPLE_HEX, 0.2)),
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
      }}
      align="center"
      justify="center"
    >
      <Text.Text
        level="h4"
        style={{
          color: color.cssString(PURPLE_HEX),
          borderTopLeftRadius: "1rem",
          borderBottomLeftRadius: "1rem",
          padding: "0.25rem 1rem",
        }}
      >
        C
      </Text.Text>
    </Flex.Box>
    <Divider.Divider y color={5} />
    <Flex.Box style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} color={9}>
        Count
      </Text.Text>
    </Flex.Box>
    <Handle.Sink location="left" id="input" />
    <Handle.Source location="right" id="output" />
  </Flex.Box>
);
