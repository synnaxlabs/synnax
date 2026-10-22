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
const REQUIRED_MESSAGES = {
  name: "Enter your name.",
  email: "Enter your work email.",
  company: "Enter your company.",
  message: "Tell us a little about your project.",
};
type Field = keyof typeof EMPTY_VALUES;
type FieldErrors = Partial<Record<Field, string>>;

const FieldError = ({
  field,
  message,
}: {
  field: Field;
  message?: string;
}): ReactElement | null =>
  message == null ? null : (
    <p
      className="foundation-contact-field-error"
      id={`foundation-contact-${field}-error`}
    >
      {message}
    </p>
  );

export const ContactForm = (): ReactElement => {
  const [values, setValues] = useState(EMPTY_VALUES);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [pending, setPending] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState(false);
  const inFlight = useRef(false);
  const formRef = useRef<HTMLFormElement>(null);
  const successRef = useRef<HTMLDivElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  const resetFocus = useRef(false);
  const invalidFieldRef = useRef<Field | undefined>(undefined);

  useEffect(() => {
    // Keep native validation for the static form until the inline handlers hydrate.
    if (formRef.current != null) formRef.current.noValidate = true;
    if (success) successRef.current?.focus();
    else if (resetFocus.current) {
      nameRef.current?.focus();
      resetFocus.current = false;
    }
  }, [success]);

  useEffect(() => {
    const field = invalidFieldRef.current;
    if (field == null) return;
    invalidFieldRef.current = undefined;
    const input = formRef.current?.querySelector<HTMLElement>(`[name="${field}"]`);
    input?.focus({ preventScroll: true });
    input?.scrollIntoView({ block: "nearest", inline: "nearest", behavior: "instant" });
  }, [fieldErrors]);

  const validate = (field: Field, value: string): string | undefined => {
    if (field === "phone") return undefined;
    if (value.trim() === "") return REQUIRED_MESSAGES[field];
    const input = formRef.current?.querySelector<
      HTMLInputElement | HTMLTextAreaElement
    >(`[name="${field}"]`);
    if (input?.validity.typeMismatch) return "Enter a valid email address.";
    if (input?.validity.tooLong) return "Keep your message under 50,000 characters.";
    return undefined;
  };

  const update = (field: Field, value: string): void => {
    setValues((previous) => ({ ...previous, [field]: value }));
    if (fieldErrors[field] != null) {
      const message = validate(field, value);
      setFieldErrors((previous) => ({ ...previous, [field]: message }));
    }
  };

  const validateOnBlur = (field: Field): void => {
    const message = validate(field, values[field]);
    setFieldErrors((previous) => ({ ...previous, [field]: message }));
  };

  const validationProps = (field: Field) => ({
    "aria-invalid": fieldErrors[field] != null || undefined,
    "aria-describedby":
      fieldErrors[field] != null ? `foundation-contact-${field}-error` : undefined,
    onBlur: () => validateOnBlur(field),
  });

  const submit = (event: SubmitEvent<HTMLFormElement>): void => {
    event.preventDefault();
    if (inFlight.current) return;
    const errors: FieldErrors = {};
    let firstInvalid: Field | undefined;
    for (const field of Object.keys(
      REQUIRED_MESSAGES,
    ) as (keyof typeof REQUIRED_MESSAGES)[]) {
      const message = validate(field, values[field]);
      if (message == null) continue;
      errors[field] = message;
      firstInvalid ??= field;
    }
    invalidFieldRef.current = firstInvalid;
    setFieldErrors(errors);
    if (firstInvalid != null) return;
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
    setFieldErrors({});
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
          ref={formRef}
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
                {...validationProps("name")}
              />
              <FieldError field="name" message={fieldErrors.name} />
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
                {...validationProps("email")}
              />
              <FieldError field="email" message={fieldErrors.email} />
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
                {...validationProps("company")}
              />
              <FieldError field="company" message={fieldErrors.company} />
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
                {...validationProps("message")}
              />
              <FieldError field="message" message={fieldErrors.message} />
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
