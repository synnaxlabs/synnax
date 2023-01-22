import { RUNTIME } from "@synnaxlabs/x";
import baseMemoize from "proxy-memoize";

export const proxyMemo: typeof baseMemoize = (fn, opts) => {
  if (RUNTIME === "browser") return baseMemoize(fn, opts);
  return fn;
};
