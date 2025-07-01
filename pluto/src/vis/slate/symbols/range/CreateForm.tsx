import { type color } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Color } from "@/color";
import { Form } from "@/form";

export const CreateForm = () => (
  <Align.Space empty>
    <Form.TextField path="range.name" />
    <Form.Field<color.Color> path="range.color">
      {(p) => <Color.Swatch {...p} variant="outlined" />}
    </Form.Field>
  </Align.Space>
);
