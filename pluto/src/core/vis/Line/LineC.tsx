import { ReactElement } from "react";

import { Optional } from "@synnaxlabs/x";

import { useVisElement } from "../Context";

import { LineProps } from "@/core/vis/Line/core";

export interface LineCProps extends Optional<Omit<LineProps, "key">, "strokeWidth"> {}

export const LineC = (props: LineCProps): ReactElement | null => {
  useVisElement<LineProps>("line", {
    ...props,
    strokeWidth: 1,
  });
  return null;
};
