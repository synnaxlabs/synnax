import z from "zod/v4";

import { Flex } from "@/flex";
import { Divider } from "@/divider";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

export const Update = () => (
  <Flex.Box pack x align="center" background={1} bordered borderColor={6} rounded={1}>
    <Flex.Box
      style={{
        backgroundColor: "var(--pluto-secondary-z-20)",
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        height: "8rem",
        padding: "0 1rem",
      }}
      align="center"
      justify="center"
    >
      <Icon.Logo.NI
        style={{
          width: "3.25rem",
          height: "3.25rem",
          color: "var(--pluto-secondary-z)",
        }}
      />
    </Flex.Box>
    <Divider.Divider y color={5} />
    <Flex.Box style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} color={9}>
        Update SystemLink Value
      </Text.Text>
    </Flex.Box>
    <Handle.Sink location="left" id="input" />
    <Handle.Source location="right" id="output" />
  </Flex.Box>
);
