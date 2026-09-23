import "./Notification.css";
import { type ComponentPropsWithRef, type ReactElement } from "react";
import { Button } from "../button";
import { Icon } from "../icon";
import { type NotificationSpec } from "./Aggregator";
export interface NotificationProps extends ComponentPropsWithRef<"div"> {
    status: NotificationSpec;
    silence: (key: string) => void;
    actions?: ReactElement | Button.ButtonProps[];
    /** Custom indicator glyph, tinted with the status variant color. */
    icon?: ReactElement<Icon.IconProps>;
}
export declare const Notification: ({ status: stat, silence, actions, icon, className, children, ...rest }: NotificationProps) => ReactElement;
//# sourceMappingURL=Notification.d.ts.map