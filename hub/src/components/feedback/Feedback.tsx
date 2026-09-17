// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Breadcrumb,
  Button,
  Dialog,
  Flex,
  Form,
  Icon,
  Input,
  Nav,
  Text,
} from "@synnaxlabs/pluto";
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
      variant="outlined"
    >
      <Icon.Feedback />
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
      <Flex.Box
        el="form"
        id="my-form"
        className="feedback-form"
        direction="y"
        style={{
          width: "800px",
          maxWidth: "100%",
        }}
        align="center"
        empty
      >
        <Nav.Bar location="top" size="5rem">
          <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
            <Breadcrumb.Breadcrumb>
              <Icon.Feedback color={9} />
              Feedback
            </Breadcrumb.Breadcrumb>
          </Nav.Bar.Start>
          <Nav.Bar.End style={{ paddingRight: "1rem" }}>
            <Button.Button variant="text" size="small" textColor={8}>
              <Icon.Close />
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
        <Flex.Box
          direction="y"
          style={{ width: "100%", padding: "4rem 3rem 2rem 3rem" }}
          gap="small"
        >
          <Form.Field<string>
            style={{ width: "100%" }}
            path="description"
            showLabel={false}
            showHelpText={false}
            grow
          >
            {(p) => (
              <Input.Text
                {...p}
                area
                maxLength={50000}
                placeholder="What can we improve?"
                autoFocus
                variant="text"
                grow
                shrink
                style={{
                  width: "100%",
                  maxHeight: 200,
                  flexBasis: 200,
                  height: "100%",
                  borderRadius: "2px",
                  fontSize: "var(--pluto-h4-size)",
                  whiteSpace: "pre-wrap",
                }}
              />
            )}
          </Form.Field>
          <Flex.Box direction="y" empty>
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
            <Text.Text level="small" color={10}>
              If you'd like a response, please include your name and email.
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
        <Nav.Bar location="bottom" size="7rem">
          <Nav.Bar.End style={{ paddingRight: "1.5rem" }}>
            <Button.Button
              gap="medium"
              variant="outlined"
              form="my-form"
              onClick={() => handleSuccessfulSubmit()}
              status={loading ? "loading" : undefined}
              disabled={softSuccess}
            >
              {softSuccess ? <Icon.Check /> : "Send"}
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </Flex.Box>
    </Form.Form>
  );
};
