import { Flex } from "@/flex";
import { Form as Core } from "@/form";

export const Form = () => (
  <Flex.Box x grow>
    <Core.NumericField path="value" grow />
    <Core.TextField path="units" grow />
  </Flex.Box>
);
