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
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { type Organization } from "@/server/db/schema";
import { DESKTOP_CHANNELS } from "@/server/license/limits";
import { post, reload } from "@/ui/api";
import { Modal } from "@/ui/modal";

const CAPS = ["default", "limited", "unlimited"] as const;
type Cap = (typeof CAPS)[number];

const schema = z.object({ cap: z.enum(CAPS), channels: z.number() }).check((ctx) => {
  const { cap, channels } = ctx.value;
  if (cap !== "limited") return;
  if (!Number.isInteger(channels) || channels < 1)
    ctx.issues.push({
      code: "custom",
      message: "Use a whole number of at least 1",
      path: ["channels"],
      input: channels,
    });
});

const path = deep.path<z.infer<typeof schema>>();

const capOf = (desktopChannels: number | null): Cap => {
  if (desktopChannels == null) return "default";
  return desktopChannels === 0 ? "unlimited" : "limited";
};

/** bodyOf reads the form into the body of the route, where "" is the default. */
const bodyOf = ({ cap, channels }: z.infer<typeof schema>): { channels: string } => {
  if (cap === "default") return { channels: "" };
  return { channels: cap === "unlimited" ? "0" : String(channels) };
};

export interface ChannelsDialogProps {
  userID: string;
  personal: Organization;
}

/** ChannelsDialog sets the channel cap of a user's Desktop licenses. Staff only. */
export const ChannelsDialog = (props: ChannelsDialogProps): ReactElement => (
  <Modal.Frame
    name="Desktop channels"
    icon={<Icon.Channel />}
    trigger={
      <Dialog.Trigger variant="outlined" hideCaret>
        <Icon.Channel />
        Desktop channels
      </Dialog.Trigger>
    }
  >
    <Content {...props} />
  </Modal.Frame>
);

const Content = ({ userID, personal }: ChannelsDialogProps): ReactElement => {
  const methods = Form.use({
    values: {
      cap: capOf(personal.desktopChannels),
      channels: personal.desktopChannels ?? DESKTOP_CHANNELS,
    },
    schema,
  });
  const cap = Form.useFieldValue(path.cap, { ctx: methods });
  return (
    <Modal.Form
      methods={methods}
      submit="Save"
      onSubmit={async (value) => {
        await post(`/api/users/${userID}/channels`, bodyOf(value));
        reload();
      }}
    >
      <Form.Field
        path={path.cap}
        label="Channels per Core"
        helpText="Each machine gets the new limit at its next renewal."
      >
        {(p) => (
          <Select.Simple<Cap> {...p} resourceName="limit">
            <Select.Item itemKey="default">Default ({DESKTOP_CHANNELS})</Select.Item>
            <Select.Item itemKey="limited">Limited</Select.Item>
            <Select.Item itemKey="unlimited">Unlimited</Select.Item>
          </Select.Simple>
        )}
      </Form.Field>
      <Form.NumericField
        path={path.channels}
        label="Channels"
        visible={cap === "limited"}
        inputProps={{ emptyValue: 0 }}
      />
    </Modal.Form>
  );
};
