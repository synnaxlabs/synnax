// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@synnaxlabs/media/dist/media.css";
import "@synnaxlabs/pluto/dist/pluto.css";

import { type ReactElement } from "react";

import { Haul } from "@/app/haul";
import { Imex } from "@/app/imex";
import { Layout } from "@/app/layout";
import { Pluto } from "@/app/pluto";
import { Range } from "@/app/range";
import { Task } from "@/app/task";
import { Tree } from "@/app/tree";
import { Vis } from "@/app/vis";
import { Errors } from "@/platform/errors";
import { Layout as PlatformLayout } from "@/platform/layout";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

export interface AppProps extends Pick<Pluto.ContextProps, "workerURL"> {}

export const App = ({ workerURL }: AppProps): ReactElement => {
  Haul.useBlockDefaultDropBehavior();
  Runtime.useExternalLinkHandler();
  return (
    <Errors.OverlayWithoutStore>
      <Session.Context>
        <Errors.OverlayWithStore>
          <Layout.Context>
            <Tree.Context>
              <Range.Context>
                <Imex.Context>
                  <Task.Context>
                    <Pluto.Context workerURL={workerURL}>
                      <Vis.Canvas>
                        <Session.Modals.Context>
                          <PlatformLayout.Window />
                        </Session.Modals.Context>
                      </Vis.Canvas>
                    </Pluto.Context>
                  </Task.Context>
                </Imex.Context>
              </Range.Context>
            </Tree.Context>
          </Layout.Context>
        </Errors.OverlayWithStore>
      </Session.Context>
    </Errors.OverlayWithoutStore>
  );
};
