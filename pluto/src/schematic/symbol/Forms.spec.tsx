// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import {
  IframeEmbedForm,
  MediaEmbedForm,
  PageEmbedForm,
} from "@/schematic/symbol/Forms";
import { Theming } from "@/theming";

const embedSchema = z.object({
  url: z.string(),
  blockCookies: z.boolean().optional(),
  pageKey: z.string().optional(),
  dimensions: z.object({ width: z.number(), height: z.number() }),
  label: z.object({
    label: z.string(),
    level: z.string().optional(),
    maxInlineSize: z.number().optional(),
    align: z.string().optional(),
    direction: z.string().optional(),
    orientation: z.string().optional(),
  }),
  orientation: z.string().optional(),
});

const embedValues: z.infer<typeof embedSchema> = {
  url: "",
  blockCookies: true,
  pageKey: "",
  dimensions: { width: 320, height: 180 },
  label: { label: "Test" },
  orientation: "left",
};

const EmbedFormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof embedSchema>({
    values: deep.copy(embedValues),
    schema: embedSchema,
  });
  return (
    <Theming.Provider>
      <Form.Form<typeof embedSchema> {...methods}>{children}</Form.Form>
    </Theming.Provider>
  );
};

describe("MediaEmbedForm", () => {
  it("should render the URL input", () => {
    const { container } = render(<MediaEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    const urlInput = container.querySelector('input[type="text"]');
    expect(urlInput).not.toBeNull();
  });

  it("should render width and height fields", () => {
    const { getByText } = render(<MediaEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    expect(getByText("Width")).toBeTruthy();
    expect(getByText("Height")).toBeTruthy();
  });

  it("should show the URL placeholder", () => {
    const { container } = render(<MediaEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    const urlInput = container.querySelector(
      'input[placeholder="http://localhost:8554/stream"]',
    );
    expect(urlInput).not.toBeNull();
  });
});

describe("IframeEmbedForm", () => {
  it("should render the URL input with iframe placeholder", () => {
    const { container } = render(<IframeEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    const urlInput = container.querySelector(
      'input[placeholder="https://grafana.local/dashboard"]',
    );
    expect(urlInput).not.toBeNull();
  });

  it("should render the Block Cookies switch", () => {
    const { getByText } = render(<IframeEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    expect(getByText("Block Cookies")).toBeTruthy();
  });

  it("should have the iframe embedding tooltip on the URL input", () => {
    const { container } = render(<IframeEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    const urlInput = container.querySelector(
      'input[title="Target URL must allow/enable iframe embedding."]',
    );
    expect(urlInput).not.toBeNull();
  });
});

describe("PageEmbedForm", () => {
  it("should render the Page input as disabled", () => {
    const { container } = render(<PageEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    const pageInput = container.querySelector("input[disabled]");
    expect(pageInput).not.toBeNull();
  });

  it("should show the page placeholder", () => {
    const { container } = render(<PageEmbedForm />, {
      wrapper: EmbedFormWrapper,
    });
    const pageInput = container.querySelector('input[placeholder="Select a page"]');
    expect(pageInput).not.toBeNull();
  });
});
