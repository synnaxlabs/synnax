import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface TrackProps extends Flex.BoxProps {}

export const Track = ({ className, ...props }: TrackProps) => (
  <Flex.Box x className={CSS(CSS.BE("timeline", "track"), className)} {...props} />
);
