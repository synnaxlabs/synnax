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

import { type ReactElement } from "react";

import { Haul } from "@/app/haul";
import { Imex } from "@/app/imex";
import { Panel } from "@/app/panel";
import { Pluto } from "@/app/pluto";
import { Range } from "@/app/range";
import { Tree } from "@/app/tree";
import { Vis } from "@/app/vis";
import { Window } from "@/app/window";
import { Task } from "@/feature/task";
import { Errors } from "@/platform/errors";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

export interface AppProps extends Pluto.ContextProps {}

export const App = ({ workerURL }: AppProps): ReactElement => {
  Haul.useBlockDefaultDropBehavior();
  Runtime.useExternalLinkHandler();
  return (
    <Errors.OverlayWithoutStore>
      <Session.Context>
        <Pluto.Context workerURL={workerURL}>
          <Errors.OverlayWithStore>
            <Panel.Context>
              <Tree.Context>
                <Range.Context>
                  <Imex.Context>
                    <Task.RegistryProvider registry={Task.REGISTRY}>
                      <Vis.Canvas>
                        <Window.Window />
                      </Vis.Canvas>
                    </Task.RegistryProvider>
                  </Imex.Context>
                </Range.Context>
              </Tree.Context>
            </Panel.Context>
          </Errors.OverlayWithStore>
        </Pluto.Context>
      </Session.Context>
    </Errors.OverlayWithoutStore>
  );
};
