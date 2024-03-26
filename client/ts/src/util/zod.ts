import { z } from "zod";


export const nullableArrayZ = <Z extends z.ZodTypeAny>(item: Z) => z.union([z.null().transform(() => []), item.array()]);