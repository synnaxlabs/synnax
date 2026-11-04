// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type PointerEvent, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { type Selectable } from "@/platform/selector/types";

export interface ItemProps {
  selectable: Selectable;
  /** A card, or a button under a group heading. */
  variant: "card" | "action";
  disabled: boolean;
  hovered: boolean;
  onSelect: () => void;
  onHover: (type: string) => void;
}

export const Item = ({
  selectable: { type, title, label, description, icon },
  variant,
  disabled,
  hovered,
  onSelect,
  onHover,
}: ItemProps): ReactElement => {
  const props = {
    className: CSS.cls(
      CSS.BEM("layout-selector", "item", variant),
      hovered && PCSS.M("hovered"),
      hovered && CSS.M("hovered"),
    ),
    full: "x",
    disabled,
    "aria-label": title,
    "data-selectable": type,
    onClick: () => onSelect(),
    onPointerMove: (e: PointerEvent) => {
      if (e.pointerType !== "touch" && !disabled) onHover(type);
    },
  } as const;
  if (variant === "action")
    return (
      <Button.Button variant="text" size="small" justify="start" {...props}>
        {label ?? title}
      </Button.Button>
    );
  return (
    <Button.Button variant="card" {...props}>
      <Icon.Tile size="large">{icon}</Icon.Tile>
      <Flex.Box
        el="span"
        y
        gap="tiny"
        className={CSS.BE("layout-selector", "card-label")}
      >
        <Text.Text el="span" weight={500}>
          {title}
        </Text.Text>
        {description != null && (
          <Text.Text
            level="small"
            color={9}
            overflow="fade"
            className={CSS.BE("layout-selector", "description")}
          >
            {description}
          </Text.Text>
        )}
      </Flex.Box>
    </Button.Button>
  );
};
