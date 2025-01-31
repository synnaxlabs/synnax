import { Icon } from "@synnaxlabs/media";
import { Text } from "@synnaxlabs/pluto";
import { type FC } from "react";

import pre from "@/components/code/Code.astro";
import Details from "@/components/details/Details.astro";
import Summary from "@/components/details/Summary.astro";
import table from "@/components/Table.astro";

interface TextFactoryProps {
  level: Text.Level;
  includeAnchor?: boolean;
}

export interface TextProps extends Omit<Text.TextProps, "level"> {}

export const textFactory = ({
  level,
  includeAnchor = false,
}: TextFactoryProps): FC<TextProps> => {
  const Component = ({ children, id, ...p }: TextProps) => (
    <Text.Text id={id} level={level} {...p}>
      {children}
      {includeAnchor && (
        <a href={`#${id}`} className="heading-anchor">
          <Icon.Link />
        </a>
      )}
    </Text.Text>
  );
  Component.displayName = `Text.${level}`;
  return Component;
};

export const mdxOverrides = {
  pre,
  table,
  h1: textFactory({ level: "h1", includeAnchor: true }),
  h2: textFactory({ level: "h2", includeAnchor: true }),
  h3: textFactory({ level: "h3", includeAnchor: true }),
  details: Details,
  summary: Summary,
};
