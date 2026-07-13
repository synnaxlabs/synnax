// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Mosaic } from "@/app/mosaic";
import { Notifications } from "@/app/notifications";
import { Auth } from "@/feature/auth";
import { Project } from "@/feature/project";

/**
 * Aux is the shell for every non-main window: a pure panel viewport with the
 * panel selector strip and no navigation chrome.
 */
export const Aux = (): ReactElement => (
  <>
    <Notifications.Feed />
    <Auth.Guard>
      <Project.Guard>
        <Flex.Box x gap="tiny" grow style={{ padding: "0 1rem 1rem 1rem" }}>
          <Mosaic.Mosaic />
        </Flex.Box>
      </Project.Guard>
    </Auth.Guard>
  </>
);
