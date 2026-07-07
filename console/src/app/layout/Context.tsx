// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { CONTEXT_MENUS, LAYOUTS } from "@/app/layout/layouts";
import { Layout } from "@/platform/layout";

export interface ContextProps extends PropsWithChildren<{}> {}

export const Context = ({ children }: ContextProps): ReactElement => (
  <Layout.RendererProvider value={LAYOUTS}>
    <Layout.ContextMenuProvider value={CONTEXT_MENUS}>
      {children}
    </Layout.ContextMenuProvider>
  </Layout.RendererProvider>
);
