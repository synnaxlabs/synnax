import { ReactElement } from "react";

import { Optional } from "@synnaxlabs/x";

import { LineProps } from "@/core/vis/Line/core";
import { useVisElement } from "@/core/vis/useVisElement";

export interface LineCProps extends Optional<Omit<LineProps, "key">, "strokeWidth"> {}

export const LineC = (props: LineCProps): ReactElement | null => {
  useVisElement<LineProps>("line", {
    ...props,
    strokeWidth: 1,
  });
  return null;
};
