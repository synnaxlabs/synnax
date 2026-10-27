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
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";
import { z } from "zod";

import { type Organization } from "@/server/db/schema";
import { ADMIN_ROLE, type Member, MEMBER_ROLE, type Sent } from "@/server/directory";
import { post, reload } from "@/ui/api";
import { date, roleName, ROLES } from "@/ui/format";
import { Modal } from "@/ui/modal";
import { Empty, Page, Section } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";
import { useAction } from "@/ui/useAction";

const MEMBER_COLUMNS = "minmax(0, 2fr) minmax(0, 2fr) 16rem 4rem";
const INVITATION_COLUMNS = "minmax(0, 4fr) 16rem 14rem 4rem";

export interface MembersProps {
  organization: Organization;
  roster: Member[];
  /** invitations are the pending invitations, shown to an admin only. */
  invitations: Sent[];
  /** userID is the signed-in user. */
  userID: string;
}

/** Members is a team's members page. Admins invite, remove, and change roles. */
export const Members = ({
  organization,
  roster,
  invitations,
  userID,
}: MembersProps): ReactElement => {
  const role = roster.find((m) => m.userID === userID)?.role;
  const admin = role === ADMIN_ROLE;
  return (
    <Page
      title="Members"
      subtitle={
        role == null
          ? organization.name
          : `${organization.name}. You are ${roleName(role).toLowerCase()}.`
      }
      actions={admin && <InviteDialog organization={organization} />}
    >
      <Roster
        organization={organization}
        roster={roster}
        invitations={invitations}
        userID={userID}
        manageable={admin}
      />
    </Page>
  );
};

export interface RosterProps extends Omit<MembersProps, "userID"> {
  /** userID is the signed-in user, when they are on the team. */
  userID?: string;
  /** manageable lets the viewer change members and cancel invitations. */
  manageable: boolean;
}

/** Roster lists a team's members and, for someone who manages it, its invitations. */
export const Roster = ({
  organization,
  roster,
  invitations,
  userID,
  manageable,
}: RosterProps): ReactElement => (
  <Flex.Box y gap="huge" full="x">
    {roster.length === 0 ? (
      <Empty
        icon={<Icon.User />}
        message="No members yet"
        description={
          invitations.length > 0
            ? "Members join when they accept an invitation."
            : "Invite the team's first admin."
        }
      />
    ) : (
      <Table columns={MEMBER_COLUMNS} head={["Name", "Email", "Role", ""]}>
        {roster.map((m) => (
          <Entry
            key={m.userID}
            organization={organization}
            member={m}
            editable={manageable && m.userID !== userID}
            self={m.userID === userID}
          />
        ))}
      </Table>
    )}
    {manageable && invitations.length > 0 && (
      <Section title="Invitations">
        <Table columns={INVITATION_COLUMNS} head={["Email", "Role", "Sent", ""]}>
          {invitations.map((i) => (
            <Invitation key={i.invitationID} organization={organization} sent={i} />
          ))}
        </Table>
      </Section>
    )}
  </Flex.Box>
);

interface EntryProps {
  organization: Organization;
  member: Member;
  /** editable lets the viewer change the member's role or remove them. */
  editable: boolean;
  self: boolean;
}

const Entry = ({
  organization,
  member: m,
  editable,
  self,
}: EntryProps): ReactElement => {
  const path = `/api/organizations/${organization.key}/members/${m.userID}`;
  const changeRole = useAction(async (role: string) => {
    await post(`${path}/role`, { role });
    reload();
  });
  const remove = useAction(async () => {
    await post(`${path}/remove`);
    reload();
  });
  const error = changeRole.error ?? remove.error;
  return (
    <Row columns={MEMBER_COLUMNS}>
      <Flex.Box y gap="tiny" className="portal-list__stack">
        <Text.Text level="p" overflow="ellipsis">
          {m.name}
          {self ? " (you)" : ""}
        </Text.Text>
        {error != null && (
          <Status.Summary variant="error" level="small" message={error} />
        )}
      </Flex.Box>
      <Text.Text level="p" color={10} overflow="ellipsis">
        {m.email}
      </Text.Text>
      {editable ? (
        <Select.Simple<string>
          resourceName="Role"
          value={m.role}
          onChange={changeRole.run}
          disabled={changeRole.loading}
        >
          {ROLES.map(({ key, name }) => (
            <Select.Item key={key} itemKey={key}>
              {name}
            </Select.Item>
          ))}
        </Select.Simple>
      ) : (
        <Text.Text level="p" color={10}>
          {roleName(m.role)}
        </Text.Text>
      )}
      <Flex.Box justify="end">
        {editable && (
          <Button.Button
            variant="text"
            size="small"
            status={remove.loading ? "loading" : "error"}
            onClick={remove.run}
            onClickDelay={1000}
            tooltip="Hold to remove from the team"
            aria-label="Remove member"
          >
            <Icon.Close />
          </Button.Button>
        )}
      </Flex.Box>
    </Row>
  );
};

interface InvitationProps {
  organization: Organization;
  sent: Sent;
}

const Invitation = ({ organization, sent }: InvitationProps): ReactElement => {
  const revoke = useAction(async () => {
    await post(
      `/api/organizations/${organization.key}/invitations/${sent.invitationID}/revoke`,
    );
    reload();
  });
  return (
    <Row columns={INVITATION_COLUMNS}>
      <Flex.Box y gap="tiny" className="portal-list__stack">
        <Text.Text level="p" overflow="ellipsis">
          {sent.email}
        </Text.Text>
        {revoke.error != null && (
          <Status.Summary variant="error" level="small" message={revoke.error} />
        )}
      </Flex.Box>
      <Text.Text level="p" color={10}>
        {roleName(sent.role)}
      </Text.Text>
      <Text.Text level="p" color={10}>
        {date(sent.sentAt)}
      </Text.Text>
      <Flex.Box justify="end">
        <Button.Button
          variant="text"
          size="small"
          status={revoke.loading ? "loading" : "error"}
          onClick={revoke.run}
          onClickDelay={1000}
          tooltip="Hold to revoke the invitation"
          aria-label="Revoke invitation"
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </Row>
  );
};

const inviteSchema = z.object({
  email: z.email("Enter an email address"),
  role: z.string(),
});

export interface InviteDialogProps {
  organization: Organization;
}

/** InviteDialog emails someone an invitation to join the team. */
export const InviteDialog = ({ organization }: InviteDialogProps): ReactElement => (
  <Modal.Frame
    name={`${organization.name}.Invite a member`}
    icon={<Icon.User />}
    trigger={
      <Dialog.Trigger variant="filled" hideCaret>
        <Icon.Add />
        Invite
      </Dialog.Trigger>
    }
  >
    <InviteContent organization={organization} />
  </Modal.Frame>
);

const InviteContent = ({ organization }: InviteDialogProps): ReactElement => {
  const methods = Form.use({
    values: { email: "", role: MEMBER_ROLE },
    schema: inviteSchema,
  });
  return (
    <Modal.Form
      methods={methods}
      submit="Send invite"
      onSubmit={async (value) => {
        await post(`/api/organizations/${organization.key}/invite`, value);
        reload();
      }}
    >
      <Text.Text level="p" color={10}>
        They get an email with a link to join. Admins manage the team; members use its
        licenses.
      </Text.Text>
      <Form.Field<string> path="email" label="Email">
        {(p) => (
          <Input.Text {...p} type="email" autoFocus placeholder="name@company.com" />
        )}
      </Form.Field>
      <Form.Field<string> path="role" label="Role">
        {(p) => (
          <Select.Simple<string> {...p} resourceName="Role">
            {ROLES.map(({ key, name }) => (
              <Select.Item key={key} itemKey={key}>
                {name}
              </Select.Item>
            ))}
          </Select.Simple>
        )}
      </Form.Field>
    </Modal.Form>
  );
};
