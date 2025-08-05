import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Form as Core } from "@/form";
import { Status } from "@/status";

export const Form = (): ReactElement => (
  <Flex.Box x grow>
    <Core.Field<status.Variant> path="variant" style={{ width: "30rem" }}>
      {({ value, onChange }) => <Status.Select value={value} onChange={onChange} />}
    </Core.Field>
    <Core.TextField path="message" grow />
  </Flex.Box>
);
