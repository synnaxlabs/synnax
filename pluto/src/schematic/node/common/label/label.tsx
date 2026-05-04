// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction, type location } from "@synnaxlabs/x";
import { type CSSProperties } from "react";

import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { type GridItem } from "@/schematic/node/common/grid/Grid";
import { Text } from "@/text";

export interface Config {
  label?: string;
  level?: Text.Level;
  orientation?: location.Location;
  direction?: direction.Direction;
  maxInlineSize?: number;
  align?: Flex.Alignment;
}

export const gridItem = (
  props?: Config,
  onChange?: ({ label }: { label: Config }) => void,
): GridItem | null => {
  if (props == null) return null;
  const {
    label,
    level = "p",
    orientation = "top",
    direction,
    align,
    maxInlineSize,
  } = props;
  if (label == null || label.length === 0) return null;
  return {
    key: "label",
    element: (
      <Text.Editable
        className={CSS(CSS.BE("symbol", "label"), CSS.dir(direction))}
        level={level}
        value={label}
        onChange={(value: string) => onChange?.({ label: { ...props, label: value } })}
        allowEmpty
        style={{ textAlign: align as CSSProperties["textAlign"], maxInlineSize }}
      />
    ),
    location: orientation,
  };
};

export const defaultConfig = (label: string): Config => ({
  label,
  level: "small",
  orientation: "top",
  maxInlineSize: 150,
  align: "center",
  direction: "x",
});
