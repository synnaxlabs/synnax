import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { Form as Core } from "@/form";
import { Status } from "@/status";

export const Form = (): ReactElement => (
  <Align.Space x grow>
    <Core.Field<status.Variant> path="variant" style={{ width: "30rem" }}>
      {({ value, onChange }) => <Status.Select value={value} onChange={onChange} />}
    </Core.Field>
    <Core.TextField path="message" grow />
  </Align.Space>
);
