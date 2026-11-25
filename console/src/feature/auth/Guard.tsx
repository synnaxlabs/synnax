// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PropsWithChildren, ReactNode } from "react";

import { Login, useCompleteBrowserLogin } from "@/feature/auth/Login";
import { Core } from "@/platform/core";
import { Session } from "@/session";

const CompleteBrowserLogin = (): null => {
  useCompleteBrowserLogin();
  return null;
};

export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const core = Session.Core.useSelectSelected();
  // An invite logs in as the invited user, so it replaces the session of the main
  // window. Each new invite starts a fresh login surface on its Core.
  const invited =
    Session.Link.useSelectInvite() != null && Session.Runtime.isMainWindow();
  const invitesReceived = Session.Link.useSelectInvitesReceived();
  return (
    <>
      <CompleteBrowserLogin />
      {core != null && Core.hasLogin(core) && !invited ? (
        children
      ) : (
        <Login key={invitesReceived} />
      )}
    </>
  );
};
