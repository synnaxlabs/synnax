// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Selector } from "@/selector";
import { LAYOUT_TYPE } from "@/table/layout";
import { useCreate } from "@/table/useCreate";

export const Selectable: Selector.Selectable = ({ layoutKey: key }) => {
  const hasCreatePermission = Access.useCreateGranted(table.TYPE_ONTOLOGY_ID);
  const create = useCreate({});
  const handleClick = useCallback(() => create({ key }), [create, key]);
  if (!hasCreatePermission) return null;
  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Table"
      icon={<Icon.Table />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useCreateGranted(table.TYPE_ONTOLOGY_ID);
