import { Header as CoreHeader, HeaderButton } from "./Header";
export type { HeaderProps } from "./Header";

type CoreHeaderType = typeof CoreHeader;

interface HeaderType extends CoreHeaderType {
  Button: typeof HeaderButton;
}

export const Header = CoreHeader as HeaderType;

Header.Button = HeaderButton;
