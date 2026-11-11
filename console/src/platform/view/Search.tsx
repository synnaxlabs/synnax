// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { plural } from "pluralize";
import { type ReactElement, type ReactPortal, useState } from "react";
import { createPortal } from "react-dom";

import { useContext, useFormContext } from "@/platform/view/context";

/**
 * Searches the resources of the view. It renders at the head of the views bar when the
 * frame puts the search there, and in place otherwise.
 */
export const Search = (): ReactElement | ReactPortal | null => {
  const { search } = useFormContext("View.Search");
  const { editable, resourceType, searchInViews, searchSlot } =
    useContext("View.Search");
  const [value, setValue] = useState("");
  const handleChange = (v: string) => {
    setValue(v);
    search(v);
  };
  if (!editable || (searchInViews && searchSlot == null)) return null;
  const input = (
    <Input.Text
      size="small"
      level="h5"
      color={9}
      startContent={startContent}
      variant="text"
      grow={searchSlot != null}
      value={value}
      onChange={handleChange}
      placeholder={`Search ${plural(resourceType)}...`}
    />
  );
  return searchSlot == null ? input : createPortal(input, searchSlot);
};

const startContent = <Icon.Search color={9} />;
