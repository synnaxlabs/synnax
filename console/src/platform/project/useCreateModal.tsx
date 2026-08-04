// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, project, status, UnexpectedError } from "@synnaxlabs/client";
import {
  Button,
  type Flux,
  Form,
  Icon,
  Input,
  Nav,
  Panel,
  Project,
  Synnax,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/platform/modals";
import { Selector } from "@/platform/selector";
import { Triggers } from "@/platform/triggers";
import { Session } from "@/session";

export const useCreateModal = Modals.create(({ close }) => {
  const client = Synnax.use();
  const dispatch = Session.useDispatch();

  const { update: createPanel } = Panel.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key } }: Flux.AfterOptimisticParams<panel.Panel>) => {
        dispatch(Session.Panel.select({ key }));
      },
      [dispatch],
    ),
  });

  const { form, save, variant } = Project.useForm({
    query: {},
    initialValues: { name: "", layout: {} },
    afterSave: ({ value }) => {
      const { key } = value();
      if (key == null) throw new UnexpectedError("Project key is null");
      dispatch(Session.Project.select(key));
      createPanel({
        name: "New Panel",
        parent: project.ontologyID(key),
        root: { variant: "leaf", tabs: [Selector.createEmptyTab()] },
      });
      close();
    },
  });

  return (
    <Modals.Frame>
      <Modals.Header icon={<Icon.Project />}>Project.Create</Modals.Header>
      <Modals.Body>
        <Form.Form<typeof Project.formSchema> {...form}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                placeholder="Project Name"
                variant="text"
                autoFocus
                level="h3"
                {...p}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Modals.Body>
      <Modals.Footer>
        <Triggers.SaveHelpText action="Create" />
        <Nav.Bar.End>
          <Button.Button
            type="submit"
            variant="filled"
            form="create-project"
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            onClick={() => save()}
            trigger={Triggers.SAVE}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.Footer>
    </Modals.Frame>
  );
});
