// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/App.css";
import "@synnaxlabs/lyra/dist/base.css";

import { Fragment, type ReactElement } from "react";

import { Analytics } from "@/app/analytics";
import { Haul } from "@/app/haul";
import { Panel } from "@/app/panel";
import { Pluto } from "@/app/pluto";
import { Range } from "@/app/range";
import { Task } from "@/app/task";
import { Tree } from "@/app/tree";
import { Vis } from "@/app/vis";
import { Window } from "@/app/window";
import { Embedded } from "@/feature/embedded";
import { Core } from "@/platform/core";
import { Errors } from "@/platform/errors";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

export interface AppProps extends Pick<Pluto.ContextProps, "workerURL"> {}

const SideEffect = (): null => {
  Haul.useBlockDefaultDropBehavior();
  Runtime.useExternalLinkHandler();
  return null;
};

const BuildContext = DESKTOP ? Embedded.Provider : Fragment;

const AnalyticsProvider = DESKTOP ? Analytics.Provider : Fragment;

const BEFORE_CLOSE = DESKTOP ? Embedded.beforeClose : undefined;

export const App = ({ workerURL }: AppProps): ReactElement => (
  <>
    <SideEffect />
    <Errors.OverlayWithoutStore>
      <BuildContext>
        <Session.Context beforeClose={BEFORE_CLOSE}>
          <AnalyticsProvider>
            <Pluto.Context workerURL={workerURL}>
              {DESKTOP ? <Analytics.Watch /> : <Core.TokenKeeper />}
              <Session.SettledProvider>
                <Errors.OverlayWithStore>
                  <Panel.Context>
                    <Tree.Context>
                      <Range.Context>
                        <Task.Context>
                          <Vis.Canvas>
                            <Window.Window />
                          </Vis.Canvas>
                        </Task.Context>
                      </Range.Context>
                    </Tree.Context>
                  </Panel.Context>
                </Errors.OverlayWithStore>
              </Session.SettledProvider>
            </Pluto.Context>
          </AnalyticsProvider>
        </Session.Context>
      </BuildContext>
    </Errors.OverlayWithoutStore>
  </>
);
