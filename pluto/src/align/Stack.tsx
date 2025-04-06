import "@/align/Stack.css";

import { toArray } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { type SpaceProps } from "@/align/Space";
import { CSS } from "@/css";

export interface StackProps extends SpaceProps {}

export const Stack = ({ className, children, ...rest }: StackProps): ReactElement => {
  const arr = toArray(children);
  return (
    <Align.Space x className={CSS.B("stack")} {...rest}>
      <div />
      {arr.map((child, i) => (
        <div key={i}>{child}</div>
      ))}
    </Align.Space>
  );
};
