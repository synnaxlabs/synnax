import { location } from "@synnaxlabs/x";
import {
  Fragment,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";

export interface GridItem {
  key: string;
  element: ReactNode;
  location: location.Outer;
}

export interface BaseGridProps extends PropsWithChildren<{}> {
  items: GridItem[];
}

interface GridElProps {
  items: GridItem[];
  loc: location.Outer;
}

const GridEl = ({ items: fItems, loc }: GridElProps): ReactElement | null => {
  const items = fItems.filter((i) => i.location === loc);
  if (items.length === 0) return null;
  return (
    <Align.Space
      direction={location.direction(loc)}
      className={CSS(CSS.BE("grid", "item"), CSS.loc(loc))}
      empty
    >
      {items.map(({ element, key }) => (
        <Fragment key={key}>{element}</Fragment>
      ))}
    </Align.Space>
  );
};

export const BaseGrid = ({ children, items }: BaseGridProps) => (
  <div className={CSS(CSS.B("symbol-grid"))}>
    <GridEl items={items} loc="top" />
    <GridEl items={items} loc="left" />
    <GridEl items={items} loc="right" />
    <GridEl items={items} loc="bottom" />
    {children}
  </div>
);
