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
import { useCallback } from "react";

import { create, LAYOUT_TYPE } from "@/lineplot/layout";
import { Selector } from "@/selector";

export const LineplotSelectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const visible = Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);
  const handleClick = useCallback(() => {
    onPlace(create({ key: layoutKey }));
  }, [onPlace, layoutKey]);

  if (!visible) return null;

  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Line Plot"
      icon={<Icon.LinePlot />}
      onClick={handleClick}
    />
  );
};
LineplotSelectable.type = LAYOUT_TYPE;
LineplotSelectable.useVisible = () =>
  Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);
