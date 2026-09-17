// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Dialog,
  Flex,
  Form,
  Icon,
  Input,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useState } from "react";
import { z } from "zod";

import { reload } from "@/portal/ui/api";
import {
  ADMIN_ROLE,
  errorMessage,
  MEMBER_ROLE,
  type Membership,
  type Organization as ClerkOrganization,
  useClerk,
  useUser,
} from "@/portal/ui/clerk";
import { Enterprise } from "@/portal/ui/Enterprise";
import * as Modal from "@/portal/ui/Modal";
import { Page, Section } from "@/portal/ui/Page";
import { Row, Table } from "@/portal/ui/Table";
import { useAction } from "@/portal/ui/useAction";
import { type Organization } from "@/server/db/schema";

export interface AccountProps {
  name: string;
  email: string;
  organizations: Organization[];
}

/** Account shows the signed-in user's profile and the teams they belong to. */
export const Account = ({ name, email, organizations }: AccountProps): ReactElement => {
  const user = useUser();
  const teams = organizations.filter((o) => o.kind === "team");
  return (
    <Page title="Account" subtitle={email}>
      <Section title="Profile" actions={user != null && <EditProfileDialog />}>
        <Flex.Box bordered rounded background={1} style={{ padding: "3rem 4rem" }}>
          <Flex.Box className="portal-facts" full="x">
            <Fact label="Name" value={user?.fullName ?? name} />
            <Fact label="Email" value={email} />
          </Flex.Box>
        </Flex.Box>
      </Section>
      {teams.length === 0 ? (
        <Section title="Organization">
          <Text.Text level="p" color={9}>
            This is a personal account. An enterprise organization shares its licenses
            and support threads with every member.
          </Text.Text>
          <Enterprise />
        </Section>
      ) : (
        <Section title={teams.length === 1 ? "Organization" : "Organizations"}>
          {teams.map((team) => (
            <Team key={team.key} organization={team} />
          ))}
        </Section>
      )}
    </Page>
  );
};

const Fact = ({ label, value }: { label: string; value: string }): ReactElement => (
  <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
    <Text.Text level="small" color={9}>
      {label}
    </Text.Text>
    <Text.Text level="p" overflow="ellipsis">
      {value}
    </Text.Text>
  </Flex.Box>
);

const profileSchema = z.object({
  firstName: z.string().trim().min(1, "Enter your first name"),
  lastName: z.string().trim().min(1, "Enter your last name"),
});

const EditProfileDialog = (): ReactElement => (
  <Modal.Frame
    name="Edit profile"
    icon={<Icon.User />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Edit />
        Edit
      </Dialog.Trigger>
    }
  >
    <EditProfileContent />
  </Modal.Frame>
);

const EditProfileContent = (): ReactElement => {
  const user = useUser();
  const { close } = Dialog.useContext();
  const methods = Form.use({
    values: { firstName: user?.firstName ?? "", lastName: user?.lastName ?? "" },
    schema: profileSchema,
  });
  const action = useAction(
    useCallback(async () => {
      if (user == null || !methods.validate()) return;
      try {
        await user.update(methods.value());
      } catch (err) {
        throw new Error(errorMessage(err), { cause: err });
      }
      close();
      await reload();
    }, [user, methods, close]),
  );
  return (
    <Form.Form<typeof profileSchema> {...methods}>
      <Modal.Body gap="medium">
        <Form.TextField
          path="firstName"
          label="First name"
          inputProps={{ autoFocus: true }}
        />
        <Form.TextField path="lastName" label="Last name" />
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
          Save
        </Button.Button>
      </Modal.Footer>
    </Form.Form>
  );
};

const MEMBER_COLUMNS = "minmax(0, 2fr) minmax(0, 2fr) 16rem 4rem";

const ROLES: { key: string; name: string }[] = [
  { key: ADMIN_ROLE, name: "Admin" },
  { key: MEMBER_ROLE, name: "Member" },
];

const roleName = (role: string): string =>
  ROLES.find((r) => r.key === role)?.name ?? role;

/** Team shows one team's members. Admins can invite, remove, and change roles. */
const Team = ({ organization }: { organization: Organization }): ReactElement => {
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
    <Flex.Box
      y
      gap="medium"
      bordered
      rounded
      background={1}
      style={{ padding: "3rem 4rem" }}
    >
      <Flex.Box x justify="between" align="center">
        <Flex.Box y gap="tiny">
          <Text.Text level="h5">{organization.name}</Text.Text>
          <Text.Text level="small" color={9}>
            {membership == null
              ? ""
              : `You are ${roleName(membership.role).toLowerCase()}`}
          </Text.Text>
        </Flex.Box>
        <Flex.Box x gap="small">
          <Button.Button variant="text" href={`/portal?org=${organization.key}`}>
            Licenses
          </Button.Button>
          {admin && org != null && <InviteDialog organization={org} onDone={refresh} />}
        </Flex.Box>
      </Flex.Box>
      {error != null && (
        <Text.Text level="small" status="error">
          {error}
        </Text.Text>
      )}
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
    </Flex.Box>
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
        <Select.Static<string, { key: string; name: string }>
          data={ROLES}
          resourceName="Role"
          value={m.role}
          onChange={changeRole.run}
          disabled={changeRole.loading}
        />
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
          licenses and support.
        </Text.Text>
        <Form.Field<string> path="email" label="Email">
          {(p) => (
            <Input.Text {...p} type="email" autoFocus placeholder="name@company.com" />
          )}
        </Form.Field>
        <Form.Field<string> path="role" label="Role">
          {(p) => (
            <Select.Static<string, { key: string; name: string }>
              {...p}
              data={ROLES}
              resourceName="Role"
            />
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
