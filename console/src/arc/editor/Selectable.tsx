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

import { useCreate } from "@/arc/editor/useCreate";
import { TYPE } from "@/arc/slice";
import { Selector } from "@/selector";

export const Selectable: Selector.Selectable = ({ layoutKey }) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const createArc = useCreate();
  const handleClick = useCallback(
    () => createArc({ key: layoutKey }),
    [createArc, layoutKey],
  );
  if (!hasCreatePermission) return null;
  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = TYPE;
Selectable.useVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
