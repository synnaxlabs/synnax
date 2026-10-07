// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import {
  type ReactElement,
  type SubmitEvent,
  useEffect,
  useRef,
  useState,
} from "react";

const ENDPOINT = "https://formspree.io/f/mgegebvq";
const EMPTY_VALUES = { name: "", email: "", company: "", phone: "", message: "" };

export const ContactForm = (): ReactElement => {
  const [values, setValues] = useState(EMPTY_VALUES);
  const [pending, setPending] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState(false);
  const inFlight = useRef(false);
  const successRef = useRef<HTMLDivElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  const resetFocus = useRef(false);

  useEffect(() => {
    if (success) successRef.current?.focus();
    else if (resetFocus.current) {
      nameRef.current?.focus();
      resetFocus.current = false;
    }
  }, [success]);

  const update = (field: keyof typeof values, value: string): void =>
    setValues((previous) => ({ ...previous, [field]: value }));

  const submit = (event: SubmitEvent<HTMLFormElement>): void => {
    event.preventDefault();
    if (inFlight.current || !event.currentTarget.reportValidity()) return;
    const data = new FormData(event.currentTarget);
    if (values.phone.trim() === "") data.delete("phone");
    inFlight.current = true;
    setPending(true);
    setError(false);
    void (async () => {
      try {
        const response = await fetch(ENDPOINT, {
          method: "POST",
          body: data,
          headers: { Accept: "application/json" },
        });
        if (!response.ok) {
          setError(true);
          return;
        }
        setSuccess(true);
      } catch {
        setError(true);
      } finally {
        inFlight.current = false;
        setPending(false);
      }
    })();
  };

  const reset = (): void => {
    setValues(EMPTY_VALUES);
    setError(false);
    resetFocus.current = true;
    setSuccess(false);
  };

  return (
    <div className="foundation-contact-form-shell">
      {success ? (
        <div
          className="foundation-contact-success"
          ref={successRef}
          role="status"
          tabIndex={-1}
        >
          <Icon.Check className="foundation-contact-success-icon" aria-hidden="true" />
          <h3>Thanks for reaching out.</h3>
          <p>Our team will be in touch to talk about what you have in mind.</p>
          <Button.Button variant="text" type="button" onClick={reset}>
            Send another message <Icon.Arrow.Right aria-hidden="true" />
          </Button.Button>
        </div>
      ) : (
        <form
          className="foundation-contact-form"
          action={ENDPOINT}
          method="POST"
          onSubmit={submit}
          aria-label="Contact our team"
          aria-busy={pending}
        >
          <input type="hidden" name="source" value="Foundation" />
          <div className="foundation-contact-fields">
            <div className="foundation-contact-field">
              <label htmlFor="foundation-contact-name">Name</label>
              <Input.Text
                ref={nameRef}
                className="foundation-contact-input"
                id="foundation-contact-name"
                name="name"
                autoComplete="name"
                required
                disabled={pending}
                value={values.name}
                onChange={(value) => update("name", value)}
              />
            </div>
            <div className="foundation-contact-field">
              <label htmlFor="foundation-contact-email">Work email</label>
              <Input.Text
                className="foundation-contact-input"
                id="foundation-contact-email"
                name="email"
                type="email"
                autoComplete="email"
                required
                disabled={pending}
                value={values.email}
                onChange={(value) => update("email", value)}
              />
            </div>
            <div className="foundation-contact-field">
              <label htmlFor="foundation-contact-company">Company</label>
              <Input.Text
                className="foundation-contact-input"
                id="foundation-contact-company"
                name="company"
                autoComplete="organization"
                required
                disabled={pending}
                value={values.company}
                onChange={(value) => update("company", value)}
              />
            </div>
            <div className="foundation-contact-field">
              <label htmlFor="foundation-contact-phone">
                Phone <span>(optional)</span>
              </label>
              <Input.Text
                className="foundation-contact-input"
                id="foundation-contact-phone"
                name="phone"
                type="tel"
                autoComplete="tel"
                disabled={pending}
                value={values.phone}
                onChange={(value) => update("phone", value)}
              />
            </div>
            <div className="foundation-contact-field foundation-contact-field-wide">
              <label htmlFor="foundation-contact-message">
                What are you working on?
              </label>
              <textarea
                className="foundation-contact-message"
                id="foundation-contact-message"
                name="message"
                placeholder="Describe your systems, the problem you’re solving, or what you want to build."
                rows={5}
                required
                maxLength={50000}
                disabled={pending}
                value={values.message}
                onChange={(event) => update("message", event.target.value)}
              />
            </div>
          </div>
          {error && (
            <p className="foundation-contact-error" role="alert">
              We couldn’t send your message. Please try again, or email{" "}
              <a href="mailto:info@synnaxlabs.com">info@synnaxlabs.com</a>.
            </p>
          )}
          <div className="foundation-contact-submit-row">
            <p className="foundation-contact-note">
              Your message goes directly to our team.
            </p>
            <Button.Button
              className="foundation-contact-submit"
              type="submit"
              variant="filled"
              color="#ffffff"
              disabled={pending}
              status={pending ? "loading" : undefined}
            >
              {pending ? "Sending" : "Talk to our team"}
              {!pending && <Icon.Arrow.Right aria-hidden="true" />}
            </Button.Button>
          </div>
        </form>
      )}
    </div>
  );
};
