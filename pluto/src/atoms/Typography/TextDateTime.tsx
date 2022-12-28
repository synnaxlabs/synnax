import { Text } from "./Text";
import type { TextProps } from "./Text";

import { timeStringFormatters, TimeStringFormatter } from "@/util/time";

export interface TextDateTimeProps extends Omit<TextProps, "children"> {
  children: number;
  format?: TimeStringFormatter;
}

export const TextDateTime = ({
  format = "shortDateTime",
  children,
  ...props
}: TextDateTimeProps): JSX.Element => {
  const formatter = timeStringFormatters[format];
  return <Text {...props}>{formatter(children)}</Text>;
};
