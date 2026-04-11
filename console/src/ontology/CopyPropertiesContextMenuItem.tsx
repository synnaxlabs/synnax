// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { type TreeContextMenuProps } from "@/ontology/service";

export const CopyPropertiesContextMenuItem = (
  props: TreeContextMenuProps,
): ReactElement | null => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  if (ids.length !== 1) return null;
  const id = ids[0];
  const { data, name } = getResource(id);
  const getText = useCallback(() => JSON.stringify(data), [data]);
  return (
    <Menu.CopyItem
      itemKey="copyData"
      text={getText}
      successMessage={`Copied properties for ${name} to clipboard`}
    >
      Copy properties
    </Menu.CopyItem>
  );
};
