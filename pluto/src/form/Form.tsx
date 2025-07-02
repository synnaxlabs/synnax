import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod/v4";

import { Context, type ContextValue } from "@/form/Context";

export const Form = <Z extends z.ZodType>({
  children,
  ...rest
}: PropsWithChildren<ContextValue<Z>>): ReactElement => (
  <Context value={rest as ContextValue}>{children}</Context>
);
