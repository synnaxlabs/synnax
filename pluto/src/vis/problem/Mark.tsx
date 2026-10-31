// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/problem/Mark.css";

import { status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type problem } from "@/vis/problem/aether";

const ICONS: Record<problem.Variant, Icon.FC> = {
  warning: Icon.Warning,
  error: Icon.Err,
};

const Glyph = ({ variant }: { variant: problem.Variant }): ReactElement => {
  const I = ICONS[variant];
  return (
    <Status.Indicator variant={variant}>
      <I />
    </Status.Indicator>
  );
};

const Entry = ({ variant, name, message, description }: problem.Problem) => (
  <Flex.Box y gap="tiny">
    {name !== "" && (
      <Text.Text level="small" color={9} weight={500}>
        {name}
      </Text.Text>
    )}
    <Text.Text level="small" status={variant} weight={500}>
      <Glyph variant={variant} />
      {message}
    </Text.Text>
    {description !== "" && (
      <Text.Text level="small" color={9}>
        {description}
      </Text.Text>
    )}
  </Flex.Box>
);

export interface MarkProps {
  problems: problem.Problem[];
  /**
   * A tile is a framed glyph inside the corner, for a plot or a log. A glyph alone fits
   * a table cell. A badge is a ringed glyph centered on the corner, for a schematic
   * value. Defaults to "tile".
   */
  variant?: "tile" | "glyph" | "badge";
}

/**
 * Marks the top right corner of its parent, which must be positioned, with the glyph of
 * the worst problem. Hovering the mark shows every problem. Renders nothing when there
 * are no problems.
 */
export const Mark = ({
  problems,
  variant = "tile",
}: MarkProps): ReactElement | null => {
  const worst = status.worst(...problems);
  if (worst == null) return null;
  const glyph = <Glyph variant={worst.variant} />;
  return (
    <Tooltip.Dialog>
      <Flex.Box y gap="medium" className={CSS.BE("problem-mark", "tooltip")}>
        {problems.map((p) => (
          <Entry key={`${p.name}-${p.message}-${p.description}`} {...p} />
        ))}
      </Flex.Box>
      <div
        role="img"
        aria-label={worst.message}
        className={CSS.cls(CSS.B("problem-mark"), CSS.BM("problem-mark", variant))}
      >
        {variant === "tile" ? <Icon.Tile size="tiny">{glyph}</Icon.Tile> : glyph}
      </div>
    </Tooltip.Dialog>
  );
};
