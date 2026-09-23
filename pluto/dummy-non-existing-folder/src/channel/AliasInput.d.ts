import { type channel } from "@synnaxlabs/client";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";
export interface AliasInputProps extends Input.TextProps {
    channel: channel.Key;
    range?: string;
    isDefault?: boolean;
    onReset?: () => void;
}
export declare const AliasInput: ({ channel, range, isDefault, onReset, ...rest }: AliasInputProps) => ReactElement;
//# sourceMappingURL=AliasInput.d.ts.map