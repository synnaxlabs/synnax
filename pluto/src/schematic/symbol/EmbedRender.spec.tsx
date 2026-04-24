// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { IframeEmbedBase, MediaEmbedBase } from "@/schematic/symbol/Symbols";
import { Theming } from "@/theming";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Theming.Provider>{children}</Theming.Provider>
);

const BASE_PROPS = {
  symbolKey: "test",
  position: { x: 0, y: 0 },
  aetherKey: "test",
  selected: false,
  onChange: vi.fn(),
  orientation: "left" as const,
  scale: 1,
};

describe("MediaEmbedBase", () => {
  it("should render an img element with the given URL", () => {
    const { container } = render(
      <MediaEmbedBase {...BASE_PROPS} url="https://example.com/image.png" />,
      { wrapper: Wrapper },
    );
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img!.src).toBe("https://example.com/image.png");
  });

  it("should render placeholder text when URL is empty", () => {
    const { getByText } = render(
      <MediaEmbedBase {...BASE_PROPS} url="" />,
      { wrapper: Wrapper },
    );
    expect(getByText("Enter a URL")).toBeTruthy();
  });

  it("should not render an img element when URL is empty", () => {
    const { container } = render(
      <MediaEmbedBase {...BASE_PROPS} url="" />,
      { wrapper: Wrapper },
    );
    expect(container.querySelector("img")).toBeNull();
  });

  it("should set width and height to 100%", () => {
    const { container } = render(
      <MediaEmbedBase {...BASE_PROPS} url="https://example.com/image.png" />,
      { wrapper: Wrapper },
    );
    const img = container.querySelector("img")!;
    expect(img.style.width).toBe("100%");
    expect(img.style.height).toBe("100%");
  });

  it("should set objectFit to contain", () => {
    const { container } = render(
      <MediaEmbedBase {...BASE_PROPS} url="https://example.com/image.png" />,
      { wrapper: Wrapper },
    );
    const img = container.querySelector("img")!;
    expect(img.style.objectFit).toBe("contain");
  });
});

describe("IframeEmbedBase", () => {
  it("should render an iframe element with the given URL", () => {
    const { container } = render(
      <IframeEmbedBase
        {...BASE_PROPS}
        url="https://en.m.wikipedia.org/wiki/Main_Page"
        blockCookies
      />,
      { wrapper: Wrapper },
    );
    const iframe = container.querySelector("iframe");
    expect(iframe).not.toBeNull();
    expect(iframe!.src).toBe("https://en.m.wikipedia.org/wiki/Main_Page");
  });

  it("should render placeholder text when URL is empty", () => {
    const { getByText } = render(
      <IframeEmbedBase {...BASE_PROPS} url="" blockCookies />,
      { wrapper: Wrapper },
    );
    expect(getByText("Enter a URL")).toBeTruthy();
  });

  it("should not render an iframe element when URL is empty", () => {
    const { container } = render(
      <IframeEmbedBase {...BASE_PROPS} url="" blockCookies />,
      { wrapper: Wrapper },
    );
    expect(container.querySelector("iframe")).toBeNull();
  });

  it("should set sandbox to allow-scripts when blockCookies is true", () => {
    const { container } = render(
      <IframeEmbedBase
        {...BASE_PROPS}
        url="https://en.m.wikipedia.org/wiki/Main_Page"
        blockCookies
      />,
      { wrapper: Wrapper },
    );
    const iframe = container.querySelector("iframe")!;
    expect(iframe.getAttribute("sandbox")).toBe("allow-scripts");
  });

  it("should set sandbox to allow-scripts allow-same-origin when blockCookies is false", () => {
    const { container } = render(
      <IframeEmbedBase
        {...BASE_PROPS}
        url="https://grafana.local/dashboard"
        blockCookies={false}
      />,
      { wrapper: Wrapper },
    );
    const iframe = container.querySelector("iframe")!;
    expect(iframe.getAttribute("sandbox")).toBe(
      "allow-scripts allow-same-origin",
    );
  });

  it("should have no border", () => {
    const { container } = render(
      <IframeEmbedBase
        {...BASE_PROPS}
        url="https://en.m.wikipedia.org/wiki/Main_Page"
        blockCookies
      />,
      { wrapper: Wrapper },
    );
    const iframe = container.querySelector("iframe")!;
    expect(iframe.getAttribute("style")).toContain("border:");
  });
});
