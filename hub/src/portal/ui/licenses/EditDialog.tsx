// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Form, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { post, reload } from "@/portal/ui/api";
import {
  checkTerms,
  TermsFields,
  termsOf,
  termsSchema,
} from "@/portal/ui/licenses/Terms";
import * as Modal from "@/portal/ui/Modal";
import { useAction } from "@/portal/ui/useAction";
import { type License } from "@/server/db/schema";

const schema = termsSchema.check(checkTerms);

export interface EditDialogProps {
  license: License;
}

/** EditDialog changes the terms of a license already issued. Staff only. */
export const EditDialog = ({ license }: EditDialogProps): ReactElement => (
  <Modal.Frame
    name={`${license.label || "License"}.Edit`}
    icon={<Icon.Edit />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Edit />
        Edit
      </Dialog.Trigger>
    }
  >
    <Content license={license} />
  </Modal.Frame>
);

const Content = ({ license }: EditDialogProps): ReactElement => {
  const { close } = Dialog.useContext();
  const methods = Form.use({ values: termsOf(license), schema });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      await post(`/api/portal/licenses/${license.key}`, methods.value());
      close();
      await reload();
    }, [methods, license.key, close]),
  );
  return (
    <Form.Form<typeof schema> {...methods}>
      <Modal.Body gap="medium">
        <TermsFields />
        <Text.Text level="small" color={9}>
          Machines already holding a seat keep their current token. Download a new one
          for each to give it the changed terms.
        </Text.Text>
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
