// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { type Component } from "@synnaxlabs/lyra/component";
import { Tag } from "@synnaxlabs/lyra/tag";
import { type ReactElement } from "react";

import { useResultMultiple } from "@/label/queries";

export interface TagsProps {
  /** The keys of the labels to show. */
  keys: label.Key[];
  /** The size of each tag. */
  size?: Component.Size;
}

/**
 * Shows a tag for each label in keys, with the name and color the label holds now.
 * Shows nothing until the labels load.
 */
export const Tags = ({ keys, size = "small" }: TagsProps): ReactElement | null => {
  const labels = useResultMultiple({ keys }).data;
  if (labels == null || labels.length === 0) return null;
  return (
    <>
      {labels.map(({ key, name, color }) => (
        <Tag.Tag key={key} color={color} size={size}>
          {name}
        </Tag.Tag>
      ))}
    </>
  );
};
