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
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type License } from "@/server/db/schema";
import { post, reload } from "@/ui/api";
import { checkTerms, TermsFields, termsOf, termsSchema } from "@/ui/licenses/Terms";
import { Modal } from "@/ui/modal";

const schema = termsSchema.check(checkTerms);

export interface EditDialogProps {
  license: License;
}

/** EditDialog changes the terms of a license already issued. Staff only. */
export const EditDialog = ({ license }: EditDialogProps): ReactElement => (
  <Modal.Frame
    name={`${license.label}.Edit`}
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
  const methods = Form.use({ values: termsOf(license), schema });
  return (
    <Modal.Form
      methods={methods}
      submit="Save"
      onSubmit={async (terms) => {
        await post(`/api/licenses/${license.key}`, terms);
        reload();
      }}
    >
      <TermsFields />
      <Text.Text level="small" color={9}>
        Machines already holding a seat keep their current license key. Download a new
        one for each to give it the changed terms.
      </Text.Text>
    </Modal.Form>
  );
};
