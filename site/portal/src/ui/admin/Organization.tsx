// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, type ReactNode } from "react";
import { z } from "zod";

import { type Activity } from "@/page";
import { type Invitation } from "@/server/db/schema";
import { type Member } from "@/server/directory";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { type Held } from "@/server/license/list";
import { type Team } from "@/server/member";
import { ActivityTable } from "@/ui/admin/ActivityTable";
import { IssueDialog } from "@/ui/admin/IssueDialog";
import { LicenseTable } from "@/ui/admin/LicenseTable";
import { navigate, post, reload } from "@/ui/api";
import { Fact, Facts } from "@/ui/Facts";
import { date, standing } from "@/ui/format";
import { Members } from "@/ui/members";
import { Modal } from "@/ui/modal";
import { Empty, Page } from "@/ui/Page";

export type OrganizationTab = "licenses" | "members" | "activity";

const TABS: { tab: OrganizationTab; label: string; path: string }[] = [
  { tab: "licenses", label: "Licenses", path: "" },
  { tab: "members", label: "Members", path: "/members" },
  { tab: "activity", label: "Activity", path: "/activity" },
];

interface LicensesTab {
  tab: "licenses";
  licenses: Held[];
  now: Date;
}

interface MembersTab {
  tab: "members";
  roster: Member[];
  invitations: Invitation[];
  now: Date;
}

interface ActivityTab extends Activity {
  tab: "activity";
}

export type OrganizationProps = { team: Team } & (
  LicensesTab | MembersTab | ActivityTab
);

/** Organization is one team's staff page: its licenses, members, and activity. */
export const Organization = (props: OrganizationProps): ReactElement => {
  const { team } = props;
  let action: ReactNode = null;
  if (props.tab === "licenses") action = <IssueDialog teams={[team]} />;
  if (props.tab === "members") action = <Members.InviteDialog organization={team} />;
  return (
    <Page
      title={team.name}
      parent={{ label: "Organizations", href: "/admin" }}
      subtitle={`Added ${date(team.createdAt)}`}
      actions={
        <>
          <RenameDialog team={team} />
          <DeleteDialog team={team} />
        </>
      }
    >
      <Flex.Box
        x
        justify="between"
        align="center"
        gap="medium"
        full="x"
        className="portal-tab-row"
      >
        <nav className="portal-subtabs" aria-label={team.name}>
          {TABS.map(({ tab, label, path }) => (
            <Button.Button
              key={tab}
              variant="text"
              size="small"
              weight={tab === props.tab ? 500 : 400}
              className="portal-tab"
              href={`/admin/organizations/${team.key}${path}`}
              aria-current={tab === props.tab ? "page" : undefined}
            >
              {label}
            </Button.Button>
          ))}
        </nav>
        {action}
      </Flex.Box>
      {props.tab === "licenses" && <Licenses {...props} />}
      {props.tab === "members" && (
        <Members.Roster
          organization={team}
          roster={props.roster}
          invitations={props.invitations}
          manageable
          now={props.now}
        />
      )}
      {props.tab === "activity" && (
        <ActivityTable
          page={props.page}
          actors={props.actors}
          olderHref={props.olderHref}
        />
      )}
    </Page>
  );
};

const Licenses = ({ licenses, now }: LicensesTab): ReactElement => {
  if (licenses.length === 0)
    return (
      <Empty
        icon={<Icon.Access />}
        message="No licenses yet"
        description="Issue one to let the team activate machines."
      />
    );
  const { active, seats, capacity, nextExpiry } = standing(licenses, now);
  return (
    <Flex.Box y gap="huge" full="x">
      <Facts>
        <Fact label="Active licenses" value={String(active)} />
        <Fact label="Machines" value={`${seats} of ${capacity} seats in use`} />
        <Fact
          label="Next expiry"
          value={nextExpiry == null ? "None" : date(nextExpiry)}
        />
      </Facts>
      <LicenseTable entries={licenses} now={now} />
    </Flex.Box>
  );
};

const renameSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name the organization")
    .max(MAX_NAME_LENGTH, "Too long"),
});

const RenameDialog = ({ team }: { team: Team }): ReactElement => (
  <Modal.Frame
    name={`${team.name}.Rename`}
    icon={<Icon.Rename />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Rename />
        Rename
      </Dialog.Trigger>
    }
  >
    <RenameContent team={team} />
  </Modal.Frame>
);

const RenameContent = ({ team }: { team: Team }): ReactElement => {
  const methods = Form.use({ values: { name: team.name }, schema: renameSchema });
  return (
    <Modal.Form
      methods={methods}
      submit="Rename"
      onSubmit={async (value) => {
        await post(`/api/organizations/${team.key}/name`, value);
        reload();
      }}
    >
      <Form.TextField path="name" label="Name" inputProps={{ autoFocus: true }} />
    </Modal.Form>
  );
};

const DeleteDialog = ({ team }: { team: Team }): ReactElement => (
  <Modal.Frame
    name={`${team.name}.Delete`}
    icon={<Icon.Delete />}
    trigger={
      <Dialog.Trigger variant="outlined" status="error" hideCaret>
        <Icon.Delete />
        Delete
      </Dialog.Trigger>
    }
  >
    <Modal.Confirm
      question={`Delete "${team.name}"?`}
      confirm="Delete"
      delay={1500}
      onConfirm={async () => {
        await post(`/api/organizations/${team.key}/delete`);
        navigate("/admin");
      }}
    >
      This deletes the team from Clerk, so its members lose access. It also deletes the
      team's licenses, their machines, its invitations, and its activity. You cannot
      undo it. A Core checks its license offline, so a Core that already holds one of
      these licenses keeps running as its terms allow.
    </Modal.Confirm>
  </Modal.Frame>
);
