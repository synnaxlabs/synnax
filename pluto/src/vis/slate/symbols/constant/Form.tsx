import { Align } from "@/align";
import { Form as Core } from "@/form";

export const Form = () => (
  <Align.Space x grow>
    <Core.NumericField path="value" grow />
    <Core.TextField path="units" grow />
  </Align.Space>
);
