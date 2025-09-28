import z from "zod";

export const nullableZ = <Z extends z.ZodType>(item: Z) =>
  z.union([
    z.union([z.null(), z.undefined()]).transform<z.infer<Z>[]>(() => []),
    item.array(),
  ]);
