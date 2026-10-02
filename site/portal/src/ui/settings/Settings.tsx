// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";
import { z } from "zod";

import { reload } from "@/ui/api";
import { type User, useUser } from "@/ui/clerk";
import { Fact, Facts } from "@/ui/Facts";
import { Modal } from "@/ui/modal";
import { Page, Section } from "@/ui/Page";

export interface SettingsProps {
  name: string;
  email: string;
}

/** Settings shows the signed-in user's profile. */
export const Settings = ({ name, email }: SettingsProps): ReactElement => {
  const user = useUser();
  return (
    <Page title="Settings" subtitle={email}>
      <Section
        title="Profile"
        actions={user != null && <EditProfileDialog user={user} />}
      >
        <Facts>
          <Fact label="Name" value={name} />
          <Fact label="Email" value={email} />
        </Facts>
      </Section>
    </Page>
  );
};

const profileSchema = z.object({
  firstName: z.string().trim().min(1, "Enter your first name"),
  lastName: z.string().trim().min(1, "Enter your last name"),
});

interface EditProfileDialogProps {
  user: User;
}

const EditProfileDialog = ({ user }: EditProfileDialogProps): ReactElement => (
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
    <EditProfileContent user={user} />
  </Modal.Frame>
);

const EditProfileContent = ({ user }: EditProfileDialogProps): ReactElement => {
  const methods = Form.use({
    values: { firstName: user.firstName ?? "", lastName: user.lastName ?? "" },
    schema: profileSchema,
  });
  return (
    <Modal.Form
      methods={methods}
      submit="Save"
      onSubmit={async (profile) => {
        await user.update(profile);
        reload();
      }}
    >
      <Form.TextField
        path="firstName"
        label="First name"
        inputProps={{ autoFocus: true }}
      />
      <Form.TextField path="lastName" label="Last name" />
    </Modal.Form>
  );
};
