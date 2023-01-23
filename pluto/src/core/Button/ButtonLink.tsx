import { TextLinkProps } from "../Typography";

import { Button, ButtonProps } from "./Button";

/** Props for the {@link ButtonLink} component. */
export interface ButtonLinkProps
  extends ButtonProps,
    Pick<TextLinkProps, "href" | "target"> {}

export const ButtonLink = ({
  href,
  target,
  results,
  ...props
}: ButtonLinkProps): JSX.Element => {
  return (
    <form action={href} target={target} rel={target}>
      <Button {...props} />
    </form>
  );
};
