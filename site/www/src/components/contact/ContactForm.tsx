// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useEffect, useRef, useState } from "react";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(1, "Please enter your name"),
  email: z.email("Please enter a valid email"),
  phone: z.string().min(1, "Please enter a phone number"),
  message: z
    .string()
    .min(1, "Please enter a message")
    .max(50000, "Message is too long"),
});

export const ContactForm = (): ReactElement => {
  const [hardSuccess, setHardSuccess] = useState(false);
  const [softSuccess, setSoftSuccess] = useState(false);
  const [loading, setLoading] = useState(false);
  const successTimeout = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => () => clearTimeout(successTimeout.current), []);

  const methods = Form.use({
    schema: formSchema,
    values: { name: "", email: "", phone: "", message: "" },
  });

  const handleSubmit = (): void => {
    void (async () => {
      if (!methods.validate()) return;
      const value = methods.value();
      const data = new FormData();
      data.append("name", value.name);
      data.append("email", value.email);
      data.append("phone", value.phone);
      data.append("message", value.message);
      setLoading(true);
      try {
        const res = await fetch("https://formspree.io/f/mgegebvq", {
          method: "POST",
          body: data,
          headers: { Accept: "application/json" },
        });
        if (res.ok) {
          setSoftSuccess(true);
          successTimeout.current = setTimeout(() => setHardSuccess(true), 500);
        } else window.alert("Something went wrong. Please try again later.");
      } catch {
        window.alert("Something went wrong. Please try again later.");
      } finally {
        setLoading(false);
      }
    })();
  };

  return (
    <div className={`contact-form-container${hardSuccess ? " success" : ""}`}>
      <Form.Form<typeof formSchema> {...methods}>
        <Flex.Box
          el="form"
          id="contact-form"
          direction="y"
          style={{ width: "100%" }}
          gap="large"
        >
          <div className="contact-form-row">
            <Form.Field<string> path="name" label="Name" align="stretch">
              {(p) => <Input.Text {...p} size="medium" placeholder="Gaal Dornik" />}
            </Form.Field>
            <Form.Field<string> path="email" label="Email" align="stretch">
              {(p) => (
                <Input.Text {...p} size="medium" placeholder="gaal@streeling.edu" />
              )}
            </Form.Field>
          </div>
          <Form.Field<string> path="phone" label="Phone" align="stretch">
            {(p) => <Input.Text {...p} size="medium" placeholder="555-555-5555" />}
          </Form.Field>
          <Form.Field<string>
            path="message"
            label="Tell us about your project"
            align="stretch"
          >
            {(p) => (
              <textarea
                className="contact-message-input"
                value={p.value}
                onChange={(e) => p.onChange(e.target.value)}
                rows={10}
              />
            )}
          </Form.Field>
          <Button.Button
            level="p"
            size="large"
            variant="filled"
            style={{ alignSelf: "flex-end" }}
            onClick={(e) => {
              e.preventDefault();
              handleSubmit();
            }}
            status={loading ? "loading" : undefined}
            disabled={loading || softSuccess}
          >
            {softSuccess && <Icon.Check />}
            Submit →
          </Button.Button>
        </Flex.Box>
        <Flex.Box
          className="contact-success-message"
          align="center"
          justify="center"
          direction="y"
          gap="large"
        >
          <Text.Text level="h3">Thanks for reaching out!</Text.Text>
          <Text.Text level="h5" className="contact-success-sub">
            We'll be in contact within 24 hours.
          </Text.Text>
          <Button.Button
            level="p"
            size="large"
            variant="text"
            onClick={() => {
              setSoftSuccess(false);
              setHardSuccess(false);
            }}
          >
            <Icon.Arrow.Left />
            Back
          </Button.Button>
        </Flex.Box>
      </Form.Form>
    </div>
  );
};
