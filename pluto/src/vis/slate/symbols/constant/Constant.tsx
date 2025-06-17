import z from "zod/v4";

import { Icon } from "@/icon";
import { Text } from "@/text";
import { Base } from "@/vis/slate/symbols/Base";
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
  <Base
    type="Constant"
    Icon={<Icon.Constant />}
    color="var(--pluto-success-z-20)"
    textColor="var(--pluto-success-z)"
    sources={[{ key: "value", Icon: Icon.Value }]}
  >
    <Text.Text level="h4" weight={500} code>
      {value.toString()}
    </Text.Text>
  </Base>
);
