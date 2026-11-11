// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, UnexpectedError, type view } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { type Flux, View as PView } from "@synnaxlabs/pluto";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Appearance } from "@/platform/appearance";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = PView.formPath;

export interface CreateModalParams {
  /** The values the view starts with, such as its type. */
  initialValues: PView.FormValues;
}

export interface CreateModalResult {
  key: view.Key;
}

const IconField = (): ReactElement => {
  const icon = Form.useField<string | undefined>(path.icon);
  const iconColor = Form.useField<color.Crude | undefined>(path.color);
  return (
    <Dialog.Frame zIndex={100}>
      <Dialog.Trigger
        hideCaret
        variant="outlined"
        size="large"
        tooltip="Change icon"
        aria-label="Change icon"
      >
        {Appearance.renderIcon({
          icon: icon.value,
          color: iconColor.value,
          fallback: Icon.View,
        })}
      </Dialog.Trigger>
      <Appearance.Dialog aria-label="View icon">
        <Appearance.Picker
          icon={icon.value}
          color={iconColor.value}
          fallback={Icon.View}
          onIconChange={icon.onChange}
          onColorChange={iconColor.onChange}
        />
      </Appearance.Dialog>
    </Dialog.Frame>
  );
};

const Create = ({
  initialValues,
  close,
}: Modals.ContentProps<CreateModalParams, CreateModalResult>): ReactElement => {
  const { form, save, variant } = PView.useForm({
    query: null,
    initialValues,
    afterSave: ({
      value,
    }: Flux.AfterSaveParams<PView.FormQuery, typeof PView.formSchema>) => {
      const { key } = value();
      if (key == null) throw new UnexpectedError("View key is null");
      close({ key });
    },
  });
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.View />}>View.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof PView.formSchema> {...form}>
          {/* The name is as tall as the icon button, so its error shows under it and
              moves nothing. */}
          <Flex.Box x align="start" gap="medium">
            <IconField />
            <Form.Field path={path.name} showLabel={false} padHelpText={false} grow>
              {(p) => (
                <Input.Text
                  placeholder="View name"
                  aria-label="View name"
                  variant="text"
                  autoFocus
                  level="h3"
                  size="large"
                  {...p}
                />
              )}
            </Form.Field>
          </Flex.Box>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            variant="filled"
            onClick={() => save()}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/** Creates a view from a name, an icon, and a color, and resolves with its key. */
export const useCreateModal = Modals.createPrompt<CreateModalResult, CreateModalParams>(
  Create,
);
