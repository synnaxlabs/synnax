// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/version/Guard.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useEffect,
} from "react";

import { Shell } from "@/feature/shell";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

interface MismatchProps {
  clientVersion: string;
  nodeVersion: string;
}

const Mismatch = ({ clientVersion, nodeVersion }: MismatchProps): ReactElement => {
  const target = Session.Core.useSelectSelected();
  const logout = Core.useLogout();
  const url =
    target == null
      ? undefined
      : `${target.secure ? "https" : "http"}://${target.host}:${target.port}`;
  return (
    <Shell.Frame className={CSS.B("version-mismatch")} connection={target}>
      <Flex.Box
        y
        align="center"
        justify="center"
        gap="huge"
        className={CSS.BE("version-mismatch", "body")}
      >
        <PlatformShell.Mark />
        <Flex.Box y align="center" gap="small" full="x">
          <Text.Text
            level="h4"
            weight={500}
            color={11}
            className={CSS.BE("version-mismatch", "text")}
          >
            {`Console ${clientVersion} cannot work with Core ${nodeVersion}`}
          </Text.Text>
          <Text.Text level="p" color={9} className={CSS.BE("version-mismatch", "text")}>
            The Console and the Core must have the same minor version.
          </Text.Text>
        </Flex.Box>
        <Flex.Box y gap="small" full="x">
          <Button.Button
            variant="filled"
            size="large"
            full="x"
            justify="center"
            href={url}
            target="_blank"
          >
            Open in browser
            <Icon.OpenExternal />
          </Button.Button>
          <Button.Button
            variant="outlined"
            size="large"
            full="x"
            justify="center"
            onClick={logout}
          >
            Choose another Core
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Shell.Frame>
  );
};

/**
 * Renders the mismatch screen instead of its children while the last check of the Core
 * reports a different minor version than the Console. The screen lifts once a check
 * reports a matching version. A browser reloads the page instead, because the Core
 * serves the Console that matches it.
 */
export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const { clientVersion, nodeVersion, clientServerCompatible } =
    Synnax.useConnectionStatus().details;
  const mismatched = nodeVersion != null && !clientServerCompatible;
  // The dev server does not come from the Core, so a reload cannot fix it.
  const reloads = Session.Runtime.ENGINE === "web" && !IS_DEV;
  useEffect(() => {
    if (mismatched && reloads) window.location.reload();
  }, [mismatched, reloads]);
  if (!mismatched) return children;
  if (reloads) return null;
  return <Mismatch clientVersion={clientVersion} nodeVersion={nodeVersion} />;
};
