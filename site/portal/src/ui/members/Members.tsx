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
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";
import { z } from "zod";

import { ADMIN_ROLE, type Member, MEMBER_ROLE } from "@/server/directory";
import { reload } from "@/ui/api";
import { type Organization, useClerk } from "@/ui/clerk";
import * as Modal from "@/ui/Modal";
import { Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";
import { useAction } from "@/ui/useAction";

const MEMBER_COLUMNS = "minmax(0, 2fr) minmax(0, 2fr) 16rem 4rem";

const ROLES: { key: string; name: string }[] = [
  { key: ADMIN_ROLE, name: "Admin" },
  { key: MEMBER_ROLE, name: "Member" },
];

const roleName = (role: string): string =>
  ROLES.find((r) => r.key === role)?.name ?? role;

/** Team reads the team's Clerk organization, the handle every change goes through. */
type Team = () => Promise<Organization>;

const useTeam = (clerkOrgID: string): Team => {
  const clerk = useClerk();
  return async () => {
    if (clerk == null) throw new Error("The page is still loading. Try again.");
    return await clerk.getOrganization(clerkOrgID);
  };
};

export interface MembersProps {
  name: string;
  clerkOrgID: string;
  roster: Member[];
  /** userID is the signed-in user. */
  userID: string;
}

/** Members lists a team's members. Admins can invite, remove, and change roles. */
export const Members = ({
  name,
  clerkOrgID,
  roster,
  userID,
}: MembersProps): ReactElement => {
  const team = useTeam(clerkOrgID);
  const role = roster.find((m) => m.userID === userID)?.role;
  const admin = role === ADMIN_ROLE;
  return (
    <Page
      title="Members"
      subtitle={
        role == null ? name : `${name}. You are ${roleName(role).toLowerCase()}.`
      }
      actions={admin && <InviteDialog name={name} team={team} />}
    >
      <Table columns={MEMBER_COLUMNS} head={["Name", "Email", "Role", ""]}>
        {roster.map((m) => (
          <Entry
            key={m.userID}
            member={m}
            team={team}
            editable={admin && m.userID !== userID}
            self={m.userID === userID}
          />
        ))}
      </Table>
    </Page>
  );
};

interface EntryProps {
  member: Member;
  team: Team;
  /** editable lets the viewer change the member's role or remove them. */
  editable: boolean;
  self: boolean;
}

const Entry = ({ member: m, team, editable, self }: EntryProps): ReactElement => {
  const changeRole = useAction(async (role: string) => {
    await (await team()).updateMember({ userId: m.userID, role });
    reload();
  });
  const remove = useAction(async () => {
    await (await team()).removeMember(m.userID);
    reload();
  });
  return (
    <Row columns={MEMBER_COLUMNS}>
      <Text.Text level="p" overflow="ellipsis">
        {m.name}
        {self ? " (you)" : ""}
      </Text.Text>
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

const inviteSchema = z.object({
  email: z.email("Enter an email address"),
  role: z.string(),
});

interface InviteDialogProps {
  name: string;
  team: Team;
}

const InviteDialog = ({ name, team }: InviteDialogProps): ReactElement => (
  <Modal.Frame
    name={`${name}.Invite a member`}
    icon={<Icon.User />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Add />
        Invite
      </Dialog.Trigger>
    }
  >
    <InviteContent team={team} />
  </Modal.Frame>
);

const InviteContent = ({ team }: { team: Team }): ReactElement => {
  const methods = Form.use({
    values: { email: "", role: MEMBER_ROLE },
    schema: inviteSchema,
  });
  return (
    <Modal.Form
      methods={methods}
      submit="Send invite"
      onSubmit={async ({ email, role }) => {
        await (await team()).inviteMember({ emailAddress: email, role });
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
