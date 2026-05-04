import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";

export const Form = (): ReactElement => (
  <Flex.Box style={{ padding: "2rem" }} align="start" x>
    <Base.Field<color.Color> path="color" label="Color" padHelpText={false}>
      {({ value, onChange, variant: _, ...rest }) => (
        <Color.Swatch value={value} onChange={onChange} {...rest} />
      )}
    </Base.Field>
  </Flex.Box>
);
