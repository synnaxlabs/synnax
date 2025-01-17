// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Breadcrumb, Form, Nav, Text } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Button } from "@synnaxlabs/pluto/button";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { Input } from "@synnaxlabs/pluto/input";
import { z } from "astro/zod";
import { type ReactElement, useState } from "react";

const formSchema = z.object({
  name: z.string().optional(),
  email: z.string().optional(),
  description: z.string().min(1, "Please enter a message"),
});

export const FeedbackButton = (): ReactElement => {
  const props = Dropdown.use();
  return (
    <Dropdown.Dialog
      className="feedback-modal"
      variant="modal"
      keepMounted={false}
      {...props}
    >
      <Button.Button
        className="feedback-button"
        variant="outlined"
        size="medium"
        iconSpacing="small"
        onClick={props.toggle}
        startIcon={<Icon.Feedback />}
      >
        Stuck? Let us know!
      </Button.Button>
      <FeedbackForm close={props.close} />
    </Dropdown.Dialog>
  );
};

interface FeedbackFormProps {
  close: () => void;
}

const FeedbackForm = ({ close }: FeedbackFormProps): ReactElement => {
  const [loading, setLoading] = useState(false);
  const [softSuccess, setSuccess] = useState(false);

  const methods = Form.use({
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
      const value = await methods.value();
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
    <Form.Form {...methods}>
      <Align.Space
        el="form"
        id="my-form"
        className="feedback-form"
        direction="y"
        style={{
          borderRadius: "1rem",
          borderColor: "var(--pluto-gray-l4)",
          width: "800px",
          maxWidth: "100%",
        }}
        bordered
        align="center"
        empty
      >
        <Nav.Bar location="top" size="5rem">
          <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
            <Breadcrumb.Breadcrumb
              level="p"
              weight={450}
              shade={7}
              icon={<Icon.Feedback />}
            >
              Feedback
            </Breadcrumb.Breadcrumb>
          </Nav.Bar.Start>
          <Nav.Bar.End style={{ paddingRight: "1rem" }}>
            <Button.Icon variant="text" size="small">
              <Icon.Close style={{ color: "var(--color-pluto-gray-l8)" }} />
            </Button.Icon>
          </Nav.Bar.End>
        </Nav.Bar>
        <Align.Space
          direction="y"
          style={{ width: "100%", padding: "4rem 3rem 2rem 3rem" }}
          size="small"
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
            <Text.Text level="small" shade={6}>
              Name and email are optional. If you'd like a response, include them and
              we'll respond within 30 min during pacific business hours.
            </Text.Text>
          </Align.Space>
        </Align.Space>
        <Nav.Bar location="bottom" size={"7rem"}>
          <Nav.Bar.End style={{ paddingRight: "1.5rem" }}>
            <Button.Button
              size="medium"
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
