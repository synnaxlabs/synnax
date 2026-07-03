// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { LAYOUT_TYPE } from "@/layered/service/arc/layout";
import { useCreate } from "@/layered/service/arc/useCreate";
import { Selector } from "@/selector";

export const Selectable: Selector.Selectable = ({ layoutKey }) => {
  const create = useCreate();
  const handleClick = useCallback(
    () => create({ key: layoutKey }),
    [create, layoutKey],
  );
  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
