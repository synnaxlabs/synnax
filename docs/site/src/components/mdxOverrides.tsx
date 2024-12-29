import pre from "@/components/code/Code.astro";
import Details from "@/components/details/Details.astro";
import Summary from "@/components/details/Summary.astro";
import table from "@/components/Table.astro";
import { Icon } from "@synnaxlabs/media";
import { Text } from "@synnaxlabs/pluto";
import { FC } from "react";

interface TextFactoryProps {
  level: Text.Level;
  includeAnchor?: boolean;
}

export const textFactory =
  ({
    level,
    includeAnchor = false,
  }: TextFactoryProps): FC<Omit<Text.TextProps, "level">> =>
  ({ children, id, ...p }) => {
    return (
      <Text.Text id={id} level={level} {...p}>
        {children}
        {includeAnchor && (
          <a href={`#${id}`} className="heading-anchor">
            <Icon.Link />
          </a>
        )}
      </Text.Text>
    );
  };

export const mdxOverrides = {
  pre,
  table,
  h1: textFactory({ level: "h1", includeAnchor: true }),
  h2: textFactory({ level: "h2", includeAnchor: true }),
  h3: textFactory({ level: "h3", includeAnchor: true }),
  h4: textFactory({ level: "h4" }),
  h5: textFactory({ level: "h5" }),
  small: textFactory({ level: "small" }),
  details: Details,
  summary: Summary,
};
