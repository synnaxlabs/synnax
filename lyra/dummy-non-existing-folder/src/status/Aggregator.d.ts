import { type CrudeTimeSpan } from "@synnaxlabs/x";
import { type PropsWithChildren } from "react";
import type z from "zod";
import { type Adder, type AsyncErrorHandler, type ErrorHandler } from "./errorHandler";
import { type Status } from "./status";
declare const useAdder: () => Adder;
export { useAdder };
/** Props for {@link Aggregator}. */
export interface AggregatorProps extends PropsWithChildren {
    /** Statuses kept before the oldest are dropped. Defaults to 500. */
    maxHistory?: number;
}
/**
 * Collects statuses from its subtree and hands them to {@link useNotifications} and
 * the status list. Mount one near the root of the app.
 */
export declare const Aggregator: ({ children, maxHistory }: AggregatorProps) => import("react").JSX.Element;
/**
 * @returns a handler that turns a caught error into an error status on the enclosing
 * {@link Aggregator}. Use it in place of `console.error` in a UI path.
 *
 * @example handleError(err, "failed to save the range");
 */
export declare const useErrorHandler: () => ErrorHandler;
/**
 * @returns a handler that runs an async function and reports a rejection as an error
 * status. Use it wherever an effect or a click handler would otherwise float a promise.
 */
export declare const useAsyncErrorHandler: () => AsyncErrorHandler;
/** A status shown as a notification, with how many identical ones it stands for. */
export type NotificationSpec<Details extends z.ZodType = z.ZodNever> = Status<Details> & {
    count: number;
};
/** Return value for {@link useNotifications}. */
export interface UseNotificationsReturn<Details extends z.ZodType = z.ZodNever> {
    statuses: NotificationSpec<Details>[];
    /** Hides one notification for good. */
    silence: (key: string) => void;
    silenceAll: () => void;
}
interface UseNotificationsProps {
    expiration?: CrudeTimeSpan;
    poll?: CrudeTimeSpan;
}
/**
 * @returns the statuses recent enough to show as notifications, with identical ones
 * folded into a single entry carrying a count. They drop off on their own after the
 * expiration.
 */
export declare const useNotifications: ({ expiration, poll, }?: UseNotificationsProps) => UseNotificationsReturn;
//# sourceMappingURL=Aggregator.d.ts.map