// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";
import { z } from "zod";

import { type Activation } from "@/server/db/schema";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { post, reload } from "@/ui/api";
import { machineName } from "@/ui/format";
import { Modal } from "@/ui/modal";

const schema = z.object({
  name: z.string().trim().min(1, "Name the machine").max(MAX_NAME_LENGTH, "Too long"),
});

export interface RenameDialogProps {
  activation: Activation;
  visible?: boolean;
  onVisibleChange?: Modal.FrameProps["onVisibleChange"];
  /** trigger opens the dialog. Leave it out to drive it from `visible`. */
  trigger?: ReactElement;
}

/** RenameDialog changes what a machine is called wherever the portal names it. */
export const RenameDialog = ({
  activation,
  visible,
  onVisibleChange,
  trigger,
}: RenameDialogProps): ReactElement => (
  <Modal.Frame
    name="Rename this machine"
    icon={<Icon.Rename />}
    visible={visible}
    onVisibleChange={onVisibleChange}
    trigger={trigger}
  >
    <Content activation={activation} />
  </Modal.Frame>
);

const Content = ({ activation }: { activation: Activation }): ReactElement => {
  const methods = Form.use({ values: { name: machineName(activation) }, schema });
  return (
    <Modal.Form
      methods={methods}
      submit="Rename"
      onSubmit={async (value) => {
        await post(`/api/activations/${activation.key}/name`, value);
        reload();
      }}
    >
      <Form.TextField
        path="name"
        label="Name"
        inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
      />
    </Modal.Form>
  );
};
