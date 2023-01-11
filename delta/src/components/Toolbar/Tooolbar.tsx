import { Header, HeaderProps, HeaderTitleProps } from "@synnaxlabs/pluto";

export const ToolbarHeader = (
  props: Omit<HeaderProps, "level" | "divided">
): JSX.Element => <Header level="h4" divided {...props} />;

export interface ToolbarTitleProps extends Pick<HeaderTitleProps, "children"> {
  icon: JSX.Element;
}

export const ToolbarTitle = ({ icon, children }: ToolbarTitleProps): JSX.Element => (
  <Header.Title startIcon={icon}>{children}</Header.Title>
);
