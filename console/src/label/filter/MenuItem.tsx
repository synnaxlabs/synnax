// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { type HasQuery } from "@/label/filter/types";
import { SelectMultiple } from "@/label/Select";
import { type View } from "@/view";

export interface MenuItemProps
  extends Pick<View.UseQueryReturn<HasQuery>, "query" | "onQueryChange"> {}

export const MenuItem = ({ query, onQueryChange }: MenuItemProps): ReactElement => {
  const handleLabelChange = useCallback(
    (labels: label.Key[]) => {
      onQueryChange((prev) => ({ ...prev, hasLabels: labels, offset: 0, limit: 0 }));
    },
    [onQueryChange],
  );
  return (
    <SelectMultiple
      value={query.hasLabels ?? defaultValue}
      onChange={handleLabelChange}
      triggerProps={triggerProps}
      location={labelLocation}
    />
  );
};

const defaultValue: label.Key[] = [];
const triggerProps = { hideTags: true, variant: "text" } as const;
const labelLocation = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
} as const;
