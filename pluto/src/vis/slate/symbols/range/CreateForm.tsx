import { color } from "@synnaxlabs/x";

import { Flex } from "@/flex";
import { Color } from "@/color";
import { Form } from "@/form";

export const CreateForm = () => (
  <Flex.Box empty>
    <Form.TextField path="range.name" />
    <Form.Field<color.Hex> path="range.color">
      {({ value, onChange }) => (
        <Color.Swatch
          value={value}
          onChange={(c) => onChange(color.hex(c))}
          variant="outlined"
        />
      )}
    </Form.Field>
  </Flex.Box>
);
