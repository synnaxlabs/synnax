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

import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { navigate, post } from "@/ui/api";
import { Modal } from "@/ui/modal";

const schema = z.object({
  recipient: z
    .string()
    .trim()
    .min(1, "Say who the link is for")
    .max(MAX_NAME_LENGTH, "Too long"),
});

/** ReferralDialog makes a referral link for one person. Staff only. */
export const ReferralDialog = (): ReactElement => (
  <Modal.Frame
    name="Make a referral link"
    icon={<Icon.Link />}
    trigger={
      <Dialog.Trigger variant="filled" hideCaret>
        <Icon.Add />
        Make a link
      </Dialog.Trigger>
    }
  >
    <Content />
  </Modal.Frame>
);

const Content = (): ReactElement => {
  const methods = Form.use({ values: { recipient: "" }, schema });
  return (
    <Modal.Form
      methods={methods}
      submit="Make the link"
      onSubmit={async (value) => {
        await post("/api/referrals", value);
        navigate("/admin/referrals");
      }}
    >
      <Form.TextField
        path="recipient"
        label="For"
        helpText="Only staff see this name."
        inputProps={{ autoFocus: true, placeholder: "Ada Lovelace" }}
      />
    </Modal.Form>
  );
};
