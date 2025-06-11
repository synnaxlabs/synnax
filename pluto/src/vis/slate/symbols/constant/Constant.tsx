import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

const stringConstant = z.object({
  type: z.literal("string"),
  value: z.string(),
});

const numericConstant = z.object({
  dataType: z.literal("float32"),
  value: z.number(),
});

export const configZ = stringConstant.or(numericConstant);

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

export const Constant = ({ value }: SymbolProps) => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        height: "8rem",
        backgroundColor: "var(--pluto-success-z-20)",
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
      }}
      align="center"
      justify="center"
    >
      <Text.Text
        level="h4"
        style={{
          color: "var(--pluto-success-z)",
          padding: "0.25rem 1rem",
        }}
      >
        C
      </Text.Text>
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Constant
      </Text.Text>
      <Text.Text level="h4" weight={500} code>
        {value.toString()}
      </Text.Text>
    </Align.Space>
    <Handle.Source location="right" id="value" />
  </Align.Pack>
);
