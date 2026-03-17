// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type agent } from "@synnaxlabs/client";
import {
  Agent as AgentFlux,
  Button,
  Form,
  Input,
  Nav,
  Synnax,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type BaseArgs, createBase } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface CreateAgentResult {
  agent: agent.Agent;
}

export interface CreateAgentArgs extends BaseArgs<CreateAgentResult> {}

export const CREATE_LAYOUT_TYPE = "agent_create_modal";

export const [useCreateModal, CreateModal] = createBase<
  CreateAgentResult,
  CreateAgentArgs
>(
  "Agent.Create",
  CREATE_LAYOUT_TYPE,
  ({ onFinish }): ReactElement => {
    const client = Synnax.use();
    const { form, save, variant } = AgentFlux.useForm({
      query: {},
      afterSave: ({ value }) => {
        const v = value();
        onFinish({
          agent: {
            key: v.key ?? "",
            name: v.name,
            messages: v.messages ?? [],
            arcKey: v.arcKey ?? "",
            state: v.state ?? "stopped",
          },
        });
      },
    });

    const footer = (
      <>
        <Triggers.SaveHelpText action="Create" trigger={Triggers.SAVE} />
        <Nav.Bar.End align="center">
          <Button.Button
            variant="filled"
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            onClick={() => save()}
            trigger={Triggers.SAVE}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer} className={CSS.B("agent-create-modal")}>
        <Form.Form<typeof AgentFlux.formSchema> {...form}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                autoFocus
                placeholder="Agent Name"
                level="h2"
                variant="text"
                selectOnFocus
                {...p}
              />
            )}
          </Form.Field>
        </Form.Form>
      </ModalContentLayout>
    );
  },
  { window: { size: { height: 350, width: 650 }, navTop: true }, icon: "Auto" },
);
