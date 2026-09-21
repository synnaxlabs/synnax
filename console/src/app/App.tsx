// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/App.css";
import "@synnaxlabs/media/dist/media.css";
import "@synnaxlabs/pluto/dist/pluto.css";

import { Fragment, type PropsWithChildren, type ReactElement } from "react";

import { Haul } from "@/app/haul";
import { Panel } from "@/app/panel";
import { Pluto } from "@/app/pluto";
import { Range } from "@/app/range";
import { Task } from "@/app/task";
import { Tree } from "@/app/tree";
import { Vis } from "@/app/vis";
import { Window } from "@/app/window";
import { Embedded } from "@/feature/embedded";
import { Errors } from "@/platform/errors";
import { Link } from "@/platform/link";
import { Runtime } from "@/platform/runtime";
import { Version } from "@/platform/version";
import { Session } from "@/session";

export interface AppProps extends Pick<Pluto.ContextProps, "workerURL"> {}

const SideEffect = (): null => {
  Haul.useBlockDefaultDropBehavior();
  Runtime.useExternalLinkHandler();
  return null;
};

const DesktopContext = ({ children }: PropsWithChildren): ReactElement => (
  <Embedded.Provider>
    <Version.InstallProvider middleware={Embedded.installMiddleware}>
      <Link.Disabled>{children}</Link.Disabled>
    </Version.InstallProvider>
  </Embedded.Provider>
);

const BuildContext = DESKTOP ? DesktopContext : Fragment;

export const App = ({ workerURL }: AppProps): ReactElement => (
  <>
    <SideEffect />
    <Errors.OverlayWithoutStore>
      <BuildContext>
        <Session.Context>
          <Pluto.Context workerURL={workerURL}>
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
        </Session.Context>
      </BuildContext>
    </Errors.OverlayWithoutStore>
  </>
);
