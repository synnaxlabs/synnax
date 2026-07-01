// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";

import { List } from "@/component/arc/list/List";
import { type Session } from "@/session";

export const EXPLORER_LAYOUT_TYPE = "arc_explorer";

export const EXPLORER_LAYOUT: Session.Layout.State = {
  key: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Arc Explorer",
  icon: "Explore",
  location: "mosaic",
};

export const Explorer = () => {
  const { data, getItem, subscribe, retrieve } = Arc.useList({});
  return (
    <List
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      retrieve={retrieve}
      enableSearch
      textIdPrefix="arc-explorer-text"
    />
  );
};
