// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, status, UnexpectedError } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Label, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = Label.createFormPath;

const Create = ({
  close,
}: Modals.ContentProps<Record<never, never>, label.Key>): ReactElement => {
  const client = Synnax.use();
  const { form, save, variant } = Label.useCreateForm({
    query: null,
    afterSave: ({ value }) => {
      const { key } = value();
      if (key == null) throw new UnexpectedError("Label key is null");
      close(key);
    },
  });
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Label />}>Label.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof Label.createFormSchema> {...form}>
          <Flex.Box x align="center" gap="medium">
            <Form.Field path={path.color} showLabel={false} padHelpText={false}>
              {(p) => <Color.Swatch aria-label="Label color" {...p} />}
            </Form.Field>
            <Form.Field path={path.name} showLabel={false} padHelpText={false} grow>
              {(p) => (
                <Input.Text
                  placeholder="Label name"
                  aria-label="Label name"
                  variant="text"
                  autoFocus
                  level="h3"
                  full="x"
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
            variant="filled"
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
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

/** Opens a dialog that creates a label, and resolves with its key. */
export const useCreateModal = Modals.createPrompt<label.Key>(Create);
