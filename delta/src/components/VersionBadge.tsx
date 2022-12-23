import { Text } from "@synnaxlabs/pluto";
import type { TextProps } from "@synnaxlabs/pluto";

import { useVersion } from "@/hooks";
import { Optional } from "@/util/types";

type VersionBadgeProps = Optional<TextProps, "level">;

export const VersionBadge = ({
  level = "p",
  ...props
}: VersionBadgeProps): JSX.Element => {
  const v = useVersion();
  return (
    <Text level={level} {...props}>
      {v}
    </Text>
  );
};
