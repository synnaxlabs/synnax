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

import {
  iframeScaleStyle,
  IframeEmbedBase,
  IframeEmbedPreview,
  MediaEmbedBase,
  MediaEmbedPreview,
  PageEmbedPreview,
} from "@/schematic/symbol/Symbols";
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
    const { getByText } = render(<MediaEmbedBase {...BASE_PROPS} url="" />, {
      wrapper: Wrapper,
    });
    expect(getByText("Enter a URL")).toBeTruthy();
  });

  it("should not render an img element when URL is empty", () => {
    const { container } = render(<MediaEmbedBase {...BASE_PROPS} url="" />, {
      wrapper: Wrapper,
    });
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
    expect(iframe.getAttribute("sandbox")).toBe("allow-scripts allow-same-origin");
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
    expect(iframe.style.borderStyle).toBe("none");
  });

  it("should apply scale transform when scale is provided", () => {
    const { container } = render(
      <IframeEmbedBase
        {...BASE_PROPS}
        url="https://grafana.local/dashboard"
        blockCookies
        scale={2}
      />,
      { wrapper: Wrapper },
    );
    const iframe = container.querySelector("iframe")!;
    expect(iframe.style.transform).toBe("scale(2)");
    expect(iframe.style.width).toBe("50%");
    expect(iframe.style.height).toBe("50%");
  });

  it("should default to scale 1 when scale is not provided", () => {
    const { container } = render(
      <IframeEmbedBase
        {...BASE_PROPS}
        url="https://grafana.local/dashboard"
        blockCookies
      />,
      { wrapper: Wrapper },
    );
    const iframe = container.querySelector("iframe")!;
    expect(iframe.style.transform).toBe("scale(1)");
    expect(iframe.style.width).toBe("100%");
    expect(iframe.style.height).toBe("100%");
  });
});

describe("MediaEmbedPreview", () => {
  it("should render", () => {
    const { container } = render(<MediaEmbedPreview />);
    expect(container.querySelector("svg")).not.toBeNull();
  });
});

describe("IframeEmbedPreview", () => {
  it("should render", () => {
    const { container } = render(<IframeEmbedPreview />);
    expect(container.querySelector("svg")).not.toBeNull();
  });
});

describe("PageEmbedPreview", () => {
  it("should render", () => {
    const { container } = render(<PageEmbedPreview />);
    expect(container.querySelector("svg")).not.toBeNull();
  });
});

describe("iframeScaleStyle", () => {
  it("should return identity styles at scale 1", () => {
    const style = iframeScaleStyle(1);
    expect(style.width).toBe("100%");
    expect(style.height).toBe("100%");
    expect(style.transform).toBe("scale(1)");
  });

  it("should compute inverse dimensions for scale > 1", () => {
    const style = iframeScaleStyle(2);
    expect(style.width).toBe("50%");
    expect(style.height).toBe("50%");
    expect(style.transform).toBe("scale(2)");
  });

  it("should compute inverse dimensions for fractional scale", () => {
    const style = iframeScaleStyle(0.5);
    expect(style.width).toBe("200%");
    expect(style.height).toBe("200%");
    expect(style.transform).toBe("scale(0.5)");
  });

  it("should always position absolutely at top-left", () => {
    const style = iframeScaleStyle(1.5);
    expect(style.position).toBe("absolute");
    expect(style.top).toBe(0);
    expect(style.left).toBe(0);
    expect(style.transformOrigin).toBe("0 0");
  });
});
