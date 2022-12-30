import { Text } from "@synnaxlabs/pluto";
import type { TextProps } from "@synnaxlabs/pluto";

import { useSelectVersion } from "../store";

import { Optional } from "@/util/types";

type VersionBadgeProps = Optional<TextProps, "level">;

export const VersionBadge = ({
  level = "p",
  ...props
}: VersionBadgeProps): JSX.Element => {
  const v = useSelectVersion();
  return (
    <Text level={level} {...props}>
      {"v" + v}
    </Text>
  );
};
