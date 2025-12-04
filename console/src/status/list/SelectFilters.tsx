// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { state } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Label } from "@/label";
import { type View } from "@/view";

export interface FilterContextMenuProps
  extends View.UseQueryReturn<status.MultiRetrieveArgs> {}

export const FilterContextMenu = ({
  query,
  onQueryChange,
}: FilterContextMenuProps): ReactElement => {
  const handleQueryChange = (setter: state.SetArg<status.MultiRetrieveArgs>) => {
    onQueryChange((prev) => {
      const next = state.executeSetter(setter, prev);
      return { ...next, offset: 0, limit: 0 };
    });
  };

  return (
    <Label.SelectMultiple
      value={query.hasLabels ?? []}
      onChange={(labels) => handleQueryChange((q) => ({ ...q, hasLabels: labels }))}
      triggerProps={{ hideTags: true, variant: "text" }}
      location={{ targetCorner: location.TOP_RIGHT, dialogCorner: location.TOP_LEFT }}
    />
  );
};

export interface FiltersProps extends View.UseQueryReturn<status.MultiRetrieveArgs> {}

export const Filters = (props: FiltersProps): ReactElement | null => (
  <Label.HasFilter {...props} isClosable />
);
