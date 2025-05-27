import z from "zod";

import { Align } from "@/align";
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
  <Align.Pack x align="center" background={1} bordered borderShade={5} rounded={0.5}>
    <Text.Text
      level="h4"
      style={{
        color: "var(--pluto-gray-l0)",
        backgroundColor: "var(--pluto-success-m1)",
        borderTopLeftRadius: "0.5rem",
        borderBottomLeftRadius: "0.5rem",
        padding: "0.25rem 1rem",
      }}
    >
      C
    </Text.Text>
    <Text.Text level="p" weight={500} style={{ padding: "0 2rem" }} code>
      {value.toString()}
    </Text.Text>
    <Handle.Source location="right" id="value" />
  </Align.Pack>
);
