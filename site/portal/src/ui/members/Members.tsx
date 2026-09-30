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
import { type ReactElement, useCallback, useEffect, useState } from "react";
import { z } from "zod";

import { type Organization } from "@/server/db/schema";
import {
  ADMIN_ROLE,
  errorMessage,
  MEMBER_ROLE,
  type Membership,
  type Organization as ClerkOrganization,
  useClerk,
  useUser,
} from "@/ui/clerk";
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

export interface MembersProps {
  organization: Organization;
}

/** Members lists a team's members. Admins can invite, remove, and change roles. */
export const Members = ({ organization }: MembersProps): ReactElement => {
  const user = useUser();
  const clerk = useClerk();
  const membership = user?.organizationMemberships.find(
    (m) => m.organization.id === organization.clerkOrgID,
  );
  const admin = membership?.role === ADMIN_ROLE;
  const [members, setMembers] = useState<Membership[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [org, setOrg] = useState<ClerkOrganization | null>(null);
  const refresh = useCallback(async () => {
    if (clerk == null || organization.clerkOrgID == null) return;
    try {
      const o = await clerk.getOrganization(organization.clerkOrgID);
      setOrg(o);
      const page = await o.getMemberships({ pageSize: 100 });
      setMembers(page.data);
    } catch (err) {
      setError(errorMessage(err));
    }
  }, [clerk, organization.clerkOrgID]);
  useEffect(() => {
    void refresh();
  }, [refresh]);
  return (
    <Page
      title="Members"
      subtitle={
        membership == null
          ? organization.name
          : `${organization.name}. You are ${roleName(membership.role).toLowerCase()}.`
      }
      actions={
        admin && org != null && <InviteDialog organization={org} onDone={refresh} />
      }
      error={error}
    >
      {members != null && (
        <Table columns={MEMBER_COLUMNS} head={["Name", "Email", "Role", ""]}>
          {members.map((m) => (
            <Member
              key={m.id}
              membership={m}
              organization={org}
              admin={admin}
              self={m.publicUserData?.userId === user?.id}
              onDone={refresh}
            />
          ))}
        </Table>
      )}
    </Page>
  );
};

interface MemberProps {
  membership: Membership;
  organization: ClerkOrganization | null;
  admin: boolean;
  self: boolean;
  onDone: () => Promise<void>;
}

const Member = ({
  membership: m,
  organization,
  admin,
  self,
  onDone,
}: MemberProps): ReactElement => {
  const data = m.publicUserData;
  const displayName =
    [data?.firstName, data?.lastName].filter((p) => p != null && p !== "").join(" ") ||
    data?.identifier ||
    "";
  const userId = data?.userId ?? "";
  const changeRole = useAction(
    useCallback(async () => {
      if (organization == null) return;
      const role = m.role === ADMIN_ROLE ? MEMBER_ROLE : ADMIN_ROLE;
      try {
        await organization.updateMember({ userId, role });
      } catch (err) {
        throw new Error(errorMessage(err), { cause: err });
      }
      await onDone();
    }, [organization, m.role, userId, onDone]),
  );
  const remove = useAction(
    useCallback(async () => {
      if (organization == null) return;
      try {
        await organization.removeMember(userId);
      } catch (err) {
        throw new Error(errorMessage(err), { cause: err });
      }
      await onDone();
    }, [organization, userId, onDone]),
  );
  const editable = admin && !self && organization != null;
  return (
    <Row columns={MEMBER_COLUMNS}>
      <Text.Text level="p" overflow="ellipsis">
        {displayName}
        {self ? " (you)" : ""}
      </Text.Text>
      <Text.Text level="p" color={10} overflow="ellipsis">
        {data?.identifier ?? ""}
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
  organization: ClerkOrganization;
  onDone: () => Promise<void>;
}

const InviteDialog = ({ organization, onDone }: InviteDialogProps): ReactElement => (
  <Modal.Frame
    name={`${organization.name}.Invite a member`}
    icon={<Icon.User />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Add />
        Invite
      </Dialog.Trigger>
    }
  >
    <InviteContent organization={organization} onDone={onDone} />
  </Modal.Frame>
);

const InviteContent = ({ organization, onDone }: InviteDialogProps): ReactElement => {
  const { close } = Dialog.useContext();
  const methods = Form.use({
    values: { email: "", role: MEMBER_ROLE },
    schema: inviteSchema,
  });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      const { email, role } = methods.value();
      try {
        await organization.inviteMember({ emailAddress: email, role });
      } catch (err) {
        throw new Error(errorMessage(err), { cause: err });
      }
      close();
      await onDone();
    }, [organization, methods, close, onDone]),
  );
  return (
    <Form.Form<typeof inviteSchema> {...methods}>
      <Modal.Body gap="medium">
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
      </Modal.Body>
      <Modal.Footer error={action.error}>
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          onClick={action.run}
          status={action.loading ? "loading" : undefined}
          trigger={["Control", "Enter"]}
          triggerIndicator
        >
          Send invite
        </Button.Button>
      </Modal.Footer>
    </Form.Form>
  );
};
