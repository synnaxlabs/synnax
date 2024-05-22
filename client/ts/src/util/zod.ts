import { z } from "zod";

export const nullableArrayZ = <Z extends z.ZodTypeAny>(item: Z) =>
  z.union([z.null().transform(() => [] as z.output<Z>[]), item.array()]);
