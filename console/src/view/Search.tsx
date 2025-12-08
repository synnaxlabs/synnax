// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Input } from "@synnaxlabs/pluto";
import { plural } from "pluralize";
import { type ReactElement, useCallback, useState } from "react";

import { useContext } from "@/view/context";

export const Search = (): ReactElement | null => {
  const { editable, resourceType, search } = useContext("View.Search");
  const [value, setValue] = useState("");
  const handleChange = useCallback(
    (v: string) => {
      setValue(v);
      search(v);
    },
    [search],
  );
  if (!editable) return null;
  return (
    <Input.Text
      size="small"
      level="h5"
      color={9}
      startContent={startContent}
      variant="text"
      value={value}
      onChange={handleChange}
      placeholder={`Search ${plural(resourceType)}...`}
    />
  );
};

const startContent = <Icon.Search color={9} />;
