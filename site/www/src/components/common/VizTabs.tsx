// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type ReactElement } from "react";

export interface VizTab {
  key: string;
  title: string;
  icon?: FC;
}

interface VizTabsProps {
  tabs: VizTab[];
  active: number;
  onSelect: (index: number) => void;
}

export const VizTabs = ({ tabs, active, onSelect }: VizTabsProps): ReactElement => (
  <div className="viz-tabs">
    {tabs.map(({ key, title, icon: TabIcon }, i) => (
      <button
        key={key}
        className={`viz-tab${i === active ? " viz-tab--active" : ""}`}
        onClick={() => onSelect(i)}
      >
        {TabIcon != null && <TabIcon />}
        {title}
      </button>
    ))}
  </div>
);
