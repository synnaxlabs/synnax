// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useEffect, useState } from "react";

import { type View, type Viewer } from "@/server/referral";
import { referralPath } from "@/shell";
import { post } from "@/ui/api";
import { Auth } from "@/ui/auth";
import { useLogOut } from "@/ui/clerk";
import { DOWNLOAD_URL } from "@/ui/links";
import { Panel } from "@/ui/Panel";
import { Tile } from "@/ui/Tile";
import { useAction } from "@/ui/useAction";

export interface InviteProps {
  /** view is what the link shows, or null for an unknown link. */
  view: View | null;
  /** installable is true when the browser runs on macOS or Windows. */
  installable: boolean;
}

/**
 * Invite is the page a referral link opens. It guides a visitor through the steps to
 * use Synnax Desktop: an account, the download, and a login from the app.
 */
export const Invite = ({ view, installable }: InviteProps): ReactElement => {
  if (view == null)
    return (
      <Auth.Card
        icon={
          <Tile>
            <Icon.Warning />
          </Tile>
        }
        title="This link does not work"
        description="Ask the person who sent it for a new one"
      />
    );
  return <Guide view={view} installable={installable} />;
};

const PITCH =
  "Connect to your hardware, record and show its data, and control it. Free for " +
  "personal use.";

interface GuideProps {
  view: View;
  installable: boolean;
}

const Guide = ({
  view: { key, referrer, viewer },
  installable,
}: GuideProps): ReactElement => {
  useEffect(() => {
    post(`/api/referrals/${key}/visit`).catch((err: unknown) =>
      console.error("failed to record the visit", err),
    );
  }, [key]);
  return (
    <Auth.Card
      icon={
        <Tile status={viewer?.installed === true ? "success" : undefined}>
          {viewer?.installed === true ? <Icon.Check /> : <Icon.Computer />}
        </Tile>
      }
      title={
        viewer?.installed === true
          ? "You're set up"
          : `${referrer} invited you to Synnax Desktop`
      }
      description={PITCH}
      footer={<Footer viewer={viewer} />}
    >
      <Panel gap={0} className="portal-summary">
        <Step
          number={1}
          done={viewer != null}
          title="Create your account"
          description={viewer == null ? "Or log in" : viewer.email}
        />
        <Step
          number={2}
          done={viewer?.installed === true}
          title="Download Synnax Desktop"
          description="For macOS on Apple silicon and Windows"
        />
        <Step
          number={3}
          done={viewer?.installed === true}
          title="Open the app and log in"
          description="Choose Log in, and your browser does the rest"
        />
      </Panel>
      <Next referral={key} viewer={viewer} installable={installable} />
    </Auth.Card>
  );
};

interface StepProps {
  number: number;
  done: boolean;
  title: string;
  description: string;
}

const Step = ({ number, done, title, description }: StepProps): ReactElement => (
  <Flex.Box x align="center" gap="medium">
    <span className={CSS.cls("portal-mark", done && "portal-mark--done")}>
      {done ? <Icon.Check /> : number}
    </span>
    <Flex.Box y gap={0.25} className="portal-summary__text">
      <Text.Text level="p" weight={500} color={11}>
        {title}
      </Text.Text>
      <Text.Text level="small" color={9}>
        {description}
      </Text.Text>
    </Flex.Box>
  </Flex.Box>
);

interface NextProps {
  /** referral is the key of the referral. */
  referral: string;
  viewer: Viewer | null;
  installable: boolean;
}

/** Next is the action of the first step that is not done. */
const Next = ({ referral, viewer, installable }: NextProps): ReactElement => {
  const path = referralPath(referral);
  if (viewer == null)
    return (
      <>
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          href={Auth.withTarget("/sign-up", path)}
        >
          Create an account
        </Button.Button>
        <Button.Button
          variant="outlined"
          size="large"
          full="x"
          justify="center"
          href={Auth.withTarget("/login", path)}
        >
          Log in
        </Button.Button>
      </>
    );
  if (viewer.installed)
    return (
      <Button.Button variant="filled" size="large" full="x" justify="center" href="/">
        Go to the portal
        <Icon.Arrow.Right />
      </Button.Button>
    );
  if (installable)
    return (
      <Button.Button
        variant="filled"
        size="large"
        full="x"
        justify="center"
        href={DOWNLOAD_URL}
      >
        <Icon.Download />
        Download Synnax Desktop
      </Button.Button>
    );
  return <MailLink referral={referral} email={viewer.email} />;
};

interface MailLinkProps {
  referral: string;
  email: string;
}

/** MailLink sends the link to a visitor on a phone, to open on their computer. */
const MailLink = ({ referral, email }: MailLinkProps): ReactElement => {
  const [sent, setSent] = useState(false);
  const send = useAction(async () => {
    await post(`/api/referrals/${referral}/mail`);
    setSent(true);
  });
  return (
    <Flex.Box y gap="small">
      <Text.Text level="small" color={9}>
        {sent
          ? `We sent the link to ${email}. Open it on your Mac or Windows computer.`
          : "Synnax Desktop installs on a Mac or Windows computer, not on this device."}
      </Text.Text>
      {send.error != null && (
        <Text.Text level="small" status="error">
          {send.error}
        </Text.Text>
      )}
      <Button.Button
        variant={sent ? "outlined" : "filled"}
        size="large"
        full="x"
        justify="center"
        onClick={send.run}
        status={send.loading ? "loading" : undefined}
      >
        <Icon.Link />
        {sent ? "Send it again" : "Email me the link"}
      </Button.Button>
    </Flex.Box>
  );
};

interface FooterProps {
  viewer: Viewer | null;
}

const Footer = ({ viewer }: FooterProps): ReactElement | null => {
  const logOut = useLogOut();
  if (viewer == null) return null;
  return (
    <Flex.Box x justify="center" align="center" gap="small" wrap>
      <Text.Text level="small" color={9}>
        {viewer.staff ? "Staff visits do not claim the link" : viewer.email}
      </Text.Text>
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        onClick={logOut}
        disabled={logOut == null}
      >
        Log out
      </Button.Button>
    </Flex.Box>
  );
};
