import { Icon } from "@synnaxlabs/media";
import { color, TimeSpan } from "@synnaxlabs/x";
import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

const BLUE_HEX = color.construct("#0066FF");

export const Interval = ({ duration }: SymbolProps) => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        height: "8rem",
        width: "4rem",
        backgroundColor: color.cssString(color.setAlpha(BLUE_HEX, 0.2)),
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
      }}
      align="center"
      justify="center"
    >
      <Icon.Time
        style={{
          width: "2.5rem",
          height: "2.5rem",
          color: color.cssString(BLUE_HEX),
        }}
      />
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Time Interval
      </Text.Text>
      <Text.Text level="h4" weight={500} code>
        {new TimeSpan(duration).toString()}
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="input" />
    <Handle.Source location="right" id="output" />
  </Align.Pack>
);
