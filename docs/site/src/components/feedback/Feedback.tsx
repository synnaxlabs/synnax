// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb, Dialog, Form, Icon, Nav, Text } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Button } from "@synnaxlabs/pluto/button";
import { Input } from "@synnaxlabs/pluto/input";
import { type ReactElement, useState } from "react";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().optional(),
  email: z.string().optional(),
  description: z.string().min(1, "Please enter a message"),
});

export const FeedbackButton = (): ReactElement => (
  <Dialog.Frame className="feedback-modal" variant="modal">
    <Dialog.Trigger
      className="feedback-button"
      size="medium"
      gap="small"
      startIcon={<Icon.Feedback />}
      variant="outlined"
    >
      Stuck? Let us know!
    </Dialog.Trigger>
    <Dialog.Dialog>
      <FeedbackForm close={close} />
    </Dialog.Dialog>
  </Dialog.Frame>
);

interface FeedbackFormProps {
  close: () => void;
}

const FeedbackForm = ({ close }: FeedbackFormProps): ReactElement => {
  const [loading, setLoading] = useState(false);
  const [softSuccess, setSuccess] = useState(false);

  const methods = Form.use<typeof formSchema>({
    schema: formSchema,
    values: {
      name: "",
      email: "",
      description: "",
    },
  });

  const handleSuccessfulSubmit = () => {
    void (async () => {
      if (!methods.validate()) return;
      const data = new FormData();
      const value = methods.value();
      data.append("name", value.name ?? "");
      data.append("email", value.email ?? "");
      data.append("description", value.description);
      setLoading(true);
      const res = await fetch("https://formspree.io/f/mpwwklbr", {
        method: "POST",
        body: data,
        headers: { Accept: "application/json" },
      });
      setLoading(false);
      if (res.ok) {
        setSuccess(true);
        setTimeout(() => {
          close();
        }, 300);
      } else {
        console.error(res);
        window.alert("Something went wrong. Please try again later");
      }
    })();
  };

  return (
    <Form.Form<typeof formSchema> {...methods}>
      <Align.Space
        el="form"
        id="my-form"
        className="feedback-form"
        direction="y"
        style={{
          width: "800px",
          maxWidth: "100%",
        }}
        background={1}
        borderShade={6}
        rounded={1}
        bordered
        align="center"
        empty
      >
        <Nav.Bar location="top" size="5rem">
          <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
            <Breadcrumb.Breadcrumb
              level="p"
              weight={450}
              shade={11}
              icon={<Icon.Feedback />}
            >
              Feedback
            </Breadcrumb.Breadcrumb>
          </Nav.Bar.Start>
          <Nav.Bar.End style={{ paddingRight: "1rem" }}>
            <Button.Icon variant="text" size="small">
              <Icon.Close style={{ color: "var(--color-pluto-gray-l10)" }} />
            </Button.Icon>
          </Nav.Bar.End>
        </Nav.Bar>
        <Align.Space
          direction="y"
          style={{ width: "100%", padding: "4rem 3rem 2rem 3rem" }}
          gap="small"
        >
          <Form.Field<string>
            style={{ width: "100%" }}
            path="description"
            showLabel={false}
            showHelpText={false}
          >
            {(p) => (
              <Input.TextArea
                {...p}
                size="medium"
                maxLength={50000}
                placeholder="What can we improve?"
                autoFocus
                rows={10}
                style={{
                  width: "100%",
                  borderRadius: "2px",
                  fontSize: "var(--pluto-h4-size)",
                  whiteSpace: "pre-wrap",
                }}
              />
            )}
          </Form.Field>
          <Align.Space direction="y" empty>
            <Form.Field<string>
              path="name"
              label="Name"
              align="stretch"
              showHelpText={false}
            >
              {(p) => <Input.Text {...p} size="medium" placeholder="Gaal Dornik" />}
            </Form.Field>
            <Form.Field<string> path="email" label="Email" showHelpText={false}>
              {(p) => (
                <Input.Text {...p} size="medium" placeholder="gaal@streeling.edu" />
              )}
            </Form.Field>
            <Text.Text level="small" shade={10}>
              If you'd like a response, please include your name and email.
            </Text.Text>
          </Align.Space>
        </Align.Space>
        <Nav.Bar location="bottom" size="7rem">
          <Nav.Bar.End style={{ paddingRight: "1.5rem" }}>
            <Button.Button
              gap="medium"
              form="my-form"
              onClick={() => handleSuccessfulSubmit()}
              startIcon={softSuccess ? <Icon.Check /> : undefined}
              loading={loading}
              disabled={loading || softSuccess}
            >
              Send
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </Align.Space>
    </Form.Form>
  );
};
