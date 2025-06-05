import { TimeSpan } from "@synnaxlabs/x";

import { Form as Core } from "@/form";
import { Input } from "@/input";

export const Form = () => (
  <Core.Field<number> path="duration">
    {({ value, onChange }) => (
      <Input.Numeric
        value={new TimeSpan(value).seconds}
        onChange={(v) => onChange(TimeSpan.seconds(v).nanoseconds)}
        endContent="S"
      />
    )}
  </Core.Field>
);
