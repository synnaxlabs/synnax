// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Errors, Panel } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect } from "react";

import { Analytics } from "@/platform/analytics";
import { Session } from "@/session";

interface ReportProps {
  panelKey: panel.Key;
  tabKey: panel.TabKey;
}

const Report = ({ panelKey, tabKey }: ReportProps): null => {
  const { screen } = Analytics.use();
  const type = Panel.useTabType({ key: panelKey, tabKey });
  useEffect(() => {
    screen(type);
  }, [type, screen]);
  return null;
};

const nothing = (): null => null;

/**
 * Reports the tab the user is looking at as a screen. It reads the focused tab of the
 * window instead of mounting inside each tab: every tab of a panel stays mounted, so a
 * focus subscription in each one would re-render two whole tabs on every switch.
 */
export const Screen = (): ReactElement | null => {
  const panelKey = Session.Panel.useSelectSelected();
  const tabKey = Session.Panel.useSelectFocusedTab();
  if (panelKey == null || tabKey == null) return null;
  return (
    // The type comes from a query, which suspends and can fail. Neither may reach the
    // app, since nothing here is on screen. A fresh Report per tab reports the switch
    // between two tabs that share a type.
    <Errors.SuspenseBoundary loading={null} FallbackComponent={nothing}>
      <Report key={tabKey} panelKey={panelKey} tabKey={tabKey} />
    </Errors.SuspenseBoundary>
  );
};
