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
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { reload } from "@/ui/api";
import { errorMessage, useUser } from "@/ui/clerk";
import { Fact, Facts } from "@/ui/Facts";
import * as Modal from "@/ui/Modal";
import { Page, Section } from "@/ui/Page";
import { useAction } from "@/ui/useAction";

export interface SettingsProps {
  name: string;
  email: string;
}

/** Settings shows the signed-in user's profile. */
export const Settings = ({ name, email }: SettingsProps): ReactElement => {
  const user = useUser();
  return (
    <Page title="Settings" subtitle={email}>
      <Section title="Profile" actions={user != null && <EditProfileDialog />}>
        <Facts>
          <Fact label="Name" value={user?.fullName ?? name} />
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
