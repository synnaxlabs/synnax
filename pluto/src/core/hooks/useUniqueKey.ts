import { useMemo } from "react";

import { nanoid } from "nanoid";

export const useUniqueKey = (override?: string): string =>
  useMemo(() => override ?? nanoid(), [override]);
