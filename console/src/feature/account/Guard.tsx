// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { SignIn } from "@/feature/account/SignIn";
import { useLink } from "@/feature/account/useLink";
import { useRenew } from "@/feature/account/useRenew";
import { Session } from "@/session";

/**
 * Renders the sign-in screen instead of its children while the embedded Core refuses
 * requests for want of a license. The connection check keeps polling, so the screen
 * dismisses on its own once the sign-in link applies a license.
 */
export const Guard = ({ children }: PropsWithChildren): ReactElement => {
  const status = Synnax.useConnectionStatus();
  const unlicensed =
    status.variant === "error" && status.details.reason === "unlicensed";
  return (
    <>
      {Session.Runtime.isMainWindow() && <SideEffect />}
      {unlicensed ? <SignIn /> : children}
    </>
  );
};

// The sign-in link and the renewal both apply tokens to the embedded Core, so they
// mount with the gate that waits on it. Tauri hands a deep link to every webview, and
// the license is one per app, so only the main window listens: a pre-render that
// answered the link would focus itself into view.
const SideEffect = (): null => {
  useLink();
  useRenew();
  return null;
};
