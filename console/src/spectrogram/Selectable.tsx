// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Selector } from "@/selector";
import { create, LAYOUT_TYPE } from "@/spectrogram/layout";

export const Selectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const handleClick = useCallback(() => {
    onPlace(create({ key: layoutKey }));
  }, [onPlace, layoutKey]);

  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Spectrogram"
      icon={<Icon.Visualize />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
