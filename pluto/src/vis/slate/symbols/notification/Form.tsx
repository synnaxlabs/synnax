import { type ReactElement } from "react";

import { Align } from "@/align";
import { Form as Core } from "@/form";
import { Status } from "@/status";

export const Form = (): ReactElement => (
  <Align.Space y>
    <Core.TextField path="message" />
    <Status.Select value="success" onChange={() => {}} />
  </Align.Space>
);
