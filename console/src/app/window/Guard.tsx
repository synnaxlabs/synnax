// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Account } from "@/feature/account";
import { Auth } from "@/feature/auth";
import { Embedded } from "@/feature/embedded";
import { License } from "@/feature/license";
import { Project } from "@/feature/project";

const ConsoleGuard = ({ children }: PropsWithChildren): ReactElement => (
  <Auth.Guard>
    <License.Guard>
      <Auth.ConnectionGuard>
        <Project.Guard>{children}</Project.Guard>
      </Auth.ConnectionGuard>
    </License.Guard>
  </Auth.Guard>
);

const DesktopGuard = ({ children }: PropsWithChildren): ReactElement => (
  <Account.Guard>
    <Embedded.Guard>
      <Project.Guard standalone>{children}</Project.Guard>
    </Embedded.Guard>
  </Account.Guard>
);

/** Holds the workspace back until the session has a licensed Core and a project. */
export const Guard = DESKTOP ? DesktopGuard : ConsoleGuard;
