import "./Avatar.css";
import { type Flex } from "@synnaxlabs/lyra/flex";
export declare const avatar: (username: string) => string;
export interface AvatarProps extends Flex.BoxProps {
    username: string;
}
export declare const Avatar: ({ username, className, square, style, size, ...rest }: AvatarProps) => import("react").JSX.Element;
//# sourceMappingURL=Avatar.d.ts.map