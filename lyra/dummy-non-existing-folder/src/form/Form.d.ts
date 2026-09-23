import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";
import { type ContextValue } from "./Context";
/**
 * Publishes a form built by {@link use} to its subtree, so every field hook and
 * component below binds to it. Spread the hook's return value onto it.
 */
export declare const Form: <Z extends z.ZodType>({ children, bind, set, get, mode, validate, validateAsync, value, has, remove, setStatus, clearStatuses, reset, setCurrentStateAsInitialValues, getStatuses, }: PropsWithChildren<ContextValue<Z>>) => ReactElement;
//# sourceMappingURL=Form.d.ts.map