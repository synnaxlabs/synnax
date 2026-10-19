// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";

import { post, reload, save } from "@/ui/api";
import { Auth } from "@/ui/auth";
import { useClerk } from "@/ui/clerk";
import { activateURL, type Linked, type Query } from "@/ui/desktop/url";
import { Panel } from "@/ui/Panel";
import { Tile } from "@/ui/Tile";
import { useAction } from "@/ui/useAction";

/** KEY_FILE is the license key's file name when the browser cannot open the app. */
const KEY_FILE = "synnax-desktop.lic";

export interface LinkProps extends Query {
  email: string;
}

/**
 * Link is the page the Desktop app opens in the browser: confirm the machine, issue
 * its license, and hand the license key back through the app's URL scheme.
 */
export const Link = ({
  state,
  fingerprint,
  name,
  email,
  problem,
}: LinkProps): ReactElement => {
  const clerk = useClerk();
  const [linked, setLinked] = useState<Linked | null>(null);
  const action = useAction(async () => {
    const res = await post<Linked>("/api/desktop/link", { fingerprint, name });
    setLinked(res);
    window.location.assign(activateURL(state, res));
  });
  // Reloading after the log out sends the visitor through the login page and back.
  const logOut = (): void => {
    void clerk?.signOut(() => {
      reload();
      return Promise.resolve();
    });
  };
  if (problem != null)
    return (
      <Auth.Card
        icon={
          <Tile status="error">
            <Icon.Warning />
          </Tile>
        }
        title="This machine cannot log in"
        description="Open Synnax Desktop and choose Log in again"
        error={problem}
      />
    );
  if (linked != null)
    return (
      <Auth.Card
        icon={
          <Tile status="success">
            <Icon.Check />
          </Tile>
        }
        title="You're logged in"
        description="Choose Open Synnax Desktop when your browser asks, then close this tab"
        footer={
          <Flex.Box x justify="center" gap="small" wrap>
            <Button.Button
              variant="text"
              size="small"
              textColor={9}
              onClick={() => save(new Blob([linked.key]), KEY_FILE)}
            >
              <Icon.Download />
              Download a license file
            </Button.Button>
            <Button.Button variant="text" size="small" textColor={9} href="/">
              Go to the portal
              <Icon.Arrow.Right />
            </Button.Button>
          </Flex.Box>
        }
      >
        <Summary name={name} email={email} />
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={() => window.location.assign(activateURL(state, linked))}
        >
          Open Synnax Desktop
          <Icon.Arrow.Right />
        </Button.Button>
      </Auth.Card>
    );
  return (
    <Auth.Card
      icon={
        <Tile>
          <Icon.Computer />
        </Tile>
      }
      title="Log in to Synnax Desktop"
      description="Check that this is your account"
      error={action.error}
      footer={
        <Text.Text level="small" color={9}>
          You can log this machine out later from the portal
        </Text.Text>
      }
    >
      <Summary name={name} email={email} />
      <Button.Button
        variant="filled"
        size="large"
        full="x"
        justify="center"
        status={action.loading ? "loading" : undefined}
        onClick={action.run}
        trigger={["Enter"]}
      >
        Continue
        <Icon.Arrow.Right />
      </Button.Button>
      <Button.Button
        variant="outlined"
        size="large"
        full="x"
        justify="center"
        onClick={logOut}
        disabled={clerk == null}
      >
        Not you? Log out
      </Button.Button>
    </Auth.Card>
  );
};

interface SummaryProps {
  name: string;
  email: string;
}

/** Summary names the account and the machine a login is for. */
const Summary = ({ name, email }: SummaryProps): ReactElement => (
  <Panel gap={0} className="portal-summary">
    <Entry icon={<Icon.User />} label="Account" value={email} />
    <Entry icon={<Icon.Computer />} label="Machine" value={name} />
  </Panel>
);

interface EntryProps {
  icon: ReactElement;
  label: string;
  value: string;
}

const Entry = ({ icon, label, value }: EntryProps): ReactElement => (
  <Flex.Box x align="center" gap="medium">
    {icon}
    <Flex.Box y gap={0.25} className="portal-summary__text">
      <Text.Text level="small" color={9}>
        {label}
      </Text.Text>
      <Text.Text level="p" color={11} overflow="ellipsis">
        {value}
      </Text.Text>
    </Flex.Box>
  </Flex.Box>
);
