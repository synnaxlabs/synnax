// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { plural } from "pluralize";
import { type ReactElement, useState } from "react";

import { Search as Base } from "@/platform/search";
import { useContext, useFormContext } from "@/platform/view/context";

export const Search = (): ReactElement | null => {
  const { search } = useFormContext("View.Search");
  const { editable, resourceType } = useContext("View.Search");
  const [value, setValue] = useState("");
  const handleChange = (v: string) => {
    setValue(v);
    search(v);
  };
  if (!editable) return null;
  return (
    <Base.Input
      value={value}
      onChange={handleChange}
      placeholder={`Search ${plural(resourceType)}...`}
    />
  );
};
