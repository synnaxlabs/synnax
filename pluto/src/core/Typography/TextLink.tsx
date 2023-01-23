import { forwardRef } from "react";

import { Text, TextProps } from "./Text";

import "./TextLink.css";

export interface TextLinkProps extends Omit<TextProps, "ref"> {}

export const TextLink = forwardRef<HTMLAnchorElement, TextProps>(
  ({ href, download, target, rel, ...props }: TextLinkProps, ref): JSX.Element => (
    <a
      className="pluto-text-link"
      ref={ref}
      href={href}
      download={download}
      target={target}
      rel={rel}
    >
      <Text className="pluto-text-link__text" {...props}></Text>
    </a>
  )
);
TextLink.displayName = "TextLink";
