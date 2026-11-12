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
import { Status } from "@synnaxlabs/lyra/status";
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
import { Version } from "@/platform/version";
import { Session } from "@/session";

const UPDATE_DOCS_URL =
  "https://docs.synnaxlabs.com/reference/enterprise/upgrade#update-the-console-and-drivers";

interface MismatchProps {
  clientVersion: string;
  nodeVersion: string;
}

const Mismatch = ({ clientVersion, nodeVersion }: MismatchProps): ReactElement => {
  const target = Session.Core.useSelectSelected();
  const logout = Core.useLogout();
  const check = Version.useCheck();
  // A breaking update moves the Console past the Core, so it cannot fix the mismatch.
  const updatable =
    check.variant === "success" &&
    check.available != null &&
    check.available.variant !== "breaking";
  const openInfo = Version.useInfoModal();
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
          {check.variant === "blocked" && (
            <Status.Summary
              className={CSS.BE("version-mismatch", "text")}
              status={check.status}
            />
          )}
          <Button.Button
            variant="text"
            size="small"
            href={UPDATE_DOCS_URL}
            target="_blank"
          >
            Learn how to update
            <Icon.OpenExternal />
          </Button.Button>
        </Flex.Box>
        <Flex.Box y gap="small" full="x">
          {updatable && (
            <Button.Button
              variant="filled"
              size="large"
              full="x"
              justify="center"
              onClick={() => openInfo()}
            >
              Update Console
            </Button.Button>
          )}
          <Button.Button
            variant={updatable ? "outlined" : "filled"}
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
 * reports a matching version. A page that the selected Core served reloads instead,
 * because the Core serves the Console that matches it.
 */
export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const { clientVersion, nodeVersion, clientServerCompatible } =
    Synnax.useConnectionStatus().details;
  const selected = Session.Core.useSelectSelected();
  const mismatched = nodeVersion != null && !clientServerCompatible;
  const reloads = selected != null && Core.servesPage(selected);
  useEffect(() => {
    if (mismatched && reloads) window.location.reload();
  }, [mismatched, reloads]);
  if (!mismatched) return children;
  if (reloads) return null;
  return <Mismatch clientVersion={clientVersion} nodeVersion={nodeVersion} />;
};
