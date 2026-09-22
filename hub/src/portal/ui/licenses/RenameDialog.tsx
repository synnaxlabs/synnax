// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Form, Icon } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { post, reload } from "@/portal/ui/api";
import { machineName } from "@/portal/ui/format";
import * as Modal from "@/portal/ui/Modal";
import { useAction } from "@/portal/ui/useAction";
import { type Activation } from "@/server/db/schema";
import { MAX_NAME_LENGTH } from "@/server/license/machine";

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
  const { close } = Dialog.useContext();
  const methods = Form.use({ values: { name: machineName(activation) }, schema });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      await post(`/api/portal/activations/${activation.key}/name`, methods.value());
      close();
      await reload();
    }, [methods, activation.key, close]),
  );
  return (
    <Form.Form<typeof schema> {...methods}>
      <Modal.Body gap="medium">
        <Form.TextField
          path="name"
          label="Name"
          inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
        />
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
          Rename
        </Button.Button>
      </Modal.Footer>
    </Form.Form>
  );
};
