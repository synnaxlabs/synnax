// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { create, LAYOUT_TYPE } from "@/lineplot/layout";
import { internalCreate, ZERO_STATE } from "@/lineplot/slice";
import { Selector } from "@/selector";

export const Selectable: Selector.Selectable = ({
  layoutKey,
  onPlace,
  onResolved,
}) => {
  const hasCreatePermission = Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID);
  const dispatch = useDispatch();
  const handleClick = useCallback(() => {
    // In panel mode create the local plot state (so it renders and later syncs to
    // the server on edit) and hand its ontology.ID back; otherwise open it as a
    // mosaic tab as before.
    if (onResolved != null) {
      dispatch(internalCreate({ ...deep.copy(ZERO_STATE), key: layoutKey }));
      onResolved({ resource: lineplot.ontologyID(layoutKey) });
    } else onPlace(create({ key: layoutKey }));
  }, [onResolved, onPlace, layoutKey, dispatch]);

  if (!hasCreatePermission) return null;

  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Line Plot"
      icon={<Icon.LinePlot />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID);
