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
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { type License } from "@/server/db/schema";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { post, reload } from "@/ui/api";
import { Modal } from "@/ui/modal";

const path = deep.path<z.infer<typeof schema>>();

const schema = z.object({
  label: z
    .string()
    .trim()
    .min(1, "Give the license a label")
    .max(MAX_NAME_LENGTH, "Too long"),
});

export interface LabelDialogProps {
  license: License;
}

/** LabelDialog changes what a license is called wherever the portal names it. */
export const LabelDialog = ({ license }: LabelDialogProps): ReactElement => (
  <Modal.Frame
    name="Rename this license"
    icon={<Icon.Rename />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Rename />
        Rename
      </Dialog.Trigger>
    }
  >
    <Content license={license} />
  </Modal.Frame>
);

const Content = ({ license }: LabelDialogProps): ReactElement => {
  const methods = Form.use({ values: { label: license.label }, schema });
  return (
    <Modal.Form
      methods={methods}
      submit="Rename"
      onSubmit={async (value) => {
        await post(`/api/licenses/${license.key}/label`, value);
        reload();
      }}
    >
      <Form.TextField
        path={path.label}
        label="Label"
        inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
      />
    </Modal.Form>
  );
};
