// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, render, type RenderResult } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { assert, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { computed, renderThemed, resolve } from "@/styletest";
import { mockBoundingClientRect } from "@/testutil";
import { theme } from "@/theme";
import { Theming } from "@/theming";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

describe("Swatch", () => {
  it("should give the focus back to the swatch when its picker closes", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Swatch value="#ff0000" onChange={() => {}} aria-label="Fill" />
      </Triggers.Provider>,
    );
    const swatch = page.getByRole("button", { name: "Fill" }).element();
    assert(swatch instanceof HTMLElement);
    swatch.focus();
    await userEvent.keyboard("{Enter}");
    const preset = page.getByRole("dialog").getByRole("button").last().element();
    assert(preset instanceof HTMLElement);
    preset.focus();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(swatch);
  });

  it("should close its picker on a second click", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Swatch value="#ff0000" onChange={() => {}} aria-label="Fill" />
      </Triggers.Provider>,
    );
    const swatch = page.getByRole("button", { name: "Fill" });
    await userEvent.click(swatch);
    await expect.element(page.getByRole("dialog")).toBeInTheDocument();
    await userEvent.click(swatch);
    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
    expect(swatch.element().getAttribute("aria-expanded")).toBe("false");
  });

  it("should draw the auto icon so it reads over a translucent fallback", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Swatch
          value={undefined}
          fallback="#00000033"
          onChange={() => {}}
          aria-label="Fill"
        />
      </Triggers.Provider>,
    );
    const icon = page
      .getByRole("button", { name: "Fill" })
      .element()
      .querySelector("svg");
    assert(icon != null);
    expect(computed(icon, "color")).toBe(resolve("var(--pluto-gray-l11)"));
  });

  it("should pick the icon over the checkerboard a translucent fill shows", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Swatch
          value={undefined}
          fallback="#00000080"
          onChange={() => {}}
          aria-label="Fill"
        />
      </Triggers.Provider>,
    );
    const icon = page
      .getByRole("button", { name: "Fill" })
      .element()
      .querySelector("svg");
    assert(icon != null);
    expect(computed(icon, "color")).toBe(resolve("var(--pluto-gray-l0)"));
  });

  describe("shadow", () => {
    const renderShadow = async (): Promise<HTMLElement> => {
      await renderThemed(
        <Triggers.Provider>
          <Color.Swatch
            variant="shadow"
            value="#ff0000"
            onChange={() => {}}
            aria-label="Fill"
          />
        </Triggers.Provider>,
      );
      const swatch = page.getByRole("button", { name: "Fill" }).element();
      assert(swatch instanceof HTMLElement);
      return swatch;
    };

    it("should draw no box around its chip at rest", async () => {
      const swatch = await renderShadow();
      expect(computed(swatch, "background-color")).toBe(resolve("transparent"));
      expect(computed(swatch, "border-top-color")).toBe(resolve("transparent"));
    });

    it("should keep its box while its picker is open", async () => {
      const swatch = await renderShadow();
      swatch.focus();
      await userEvent.keyboard("{Enter}");
      await expect.element(page.getByRole("dialog")).toBeInTheDocument();
      expect(swatch.getAttribute("aria-expanded")).toBe("true");
      expect(computed(swatch, "border-top-color")).toBe(
        resolve("var(--pluto-gray-l7)"),
      );
    });
  });
});

describe("Swatch", () => {
  const RED = "#ff0000";
  const GREEN = "00ff00";
  const BLUE = "0000ff";

  const SWATCH_VAR = CSS.variable("swatch", "color");
  const TEXT_VAR = CSS.variable("swatch", "text");

  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Triggers.Provider>
      <Theming.Provider>{children}</Theming.Provider>
    </Triggers.Provider>
  );

  const swatchOf = (c: RenderResult): HTMLElement => {
    const el = c.container.querySelector<HTMLElement>(`.${CSS.B("color-swatch")}`);
    if (el == null) throw new Error("no swatch rendered");
    return el;
  };

  const hexInputOf = (c: RenderResult): HTMLInputElement =>
    c.getByLabelText("Hex") as HTMLInputElement;

  const pick = (c: RenderResult, hex: string): void => {
    fireEvent.change(hexInputOf(c), { target: { value: hex } });
  };

  const closePicker = (c: RenderResult): void => {
    fireEvent.keyDown(c.container, { code: "Escape" });
  };

  describe("onChange", () => {
    it("should call onChange on every picker change by default", () => {
      const onChange = vi.fn();
      const c = render(<Color.Swatch value={RED} onChange={onChange} />, {
        wrapper: Wrapper,
      });
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      pick(c, BLUE);
      expect(onChange).toHaveBeenCalledTimes(2);
      expect(color.hex(onChange.mock.calls[1][0])).toEqual(`#${BLUE}`);
    });
  });

  describe("fallback", () => {
    it("should show the fallback marked as auto while the value is absent", () => {
      const c = render(<Color.Swatch fallback={RED} onChange={vi.fn()} />, {
        wrapper: Wrapper,
      });
      expect(swatchOf(c).className).toContain(CSS.M("auto"));
      expect(swatchOf(c).style.getPropertyValue(SWATCH_VAR)).toEqual(
        color.cssString(RED),
      );
    });

    it("should show the auto icon while the value is absent", () => {
      const c = render(<Color.Swatch fallback={RED} onChange={vi.fn()} />, {
        wrapper: Wrapper,
      });
      expect(swatchOf(c).querySelector(`.${CSS.B("icon")}`)).not.toBeNull();
    });

    it.each([theme.SYNNAX_LIGHT, theme.SYNNAX_DARK])(
      "should draw the auto icon of a clear swatch in the icon gray of $key",
      (spec) => {
        const { key, colors } = theme.themeZ.parse(spec);
        const c = render(
          <Color.Swatch fallback={color.TRANSPARENT} onChange={vi.fn()} />,
          {
            wrapper: ({ children }) => (
              <Triggers.Provider>
                <Theming.Provider theme={{ key }}>{children}</Theming.Provider>
              </Triggers.Provider>
            ),
          },
        );
        expect(swatchOf(c).style.getPropertyValue(TEXT_VAR)).toEqual(
          color.cssString(colors.gray.l9),
        );
      },
    );

    it("should hide the auto icon on a tiny swatch", () => {
      const c = render(<Color.Swatch fallback={RED} onChange={vi.fn()} size="tiny" />, {
        wrapper: Wrapper,
      });
      expect(swatchOf(c).className).toContain(CSS.M("auto"));
      expect(swatchOf(c).querySelector(`.${CSS.B("icon")}`)).toBeNull();
    });

    it("should not mark a picked value as auto", () => {
      const c = render(
        <Color.Swatch value={`#${BLUE}`} fallback={RED} onChange={vi.fn()} />,
        { wrapper: Wrapper },
      );
      expect(swatchOf(c).className).not.toContain(CSS.M("auto"));
    });

    it("should call onChange with undefined when the user picks Auto", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={`#${BLUE}`} fallback={RED} onChange={onChange} />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      fireEvent.click(c.getByLabelText("Auto"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
    });

    it("should hold a pick of Auto until the picker closes", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch
          value={`#${BLUE}`}
          fallback={RED}
          onChange={onChange}
          onlyChangeOnBlur
        />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      fireEvent.click(c.getByLabelText("Auto"));
      expect(onChange).not.toHaveBeenCalled();
      expect(swatchOf(c).className).toContain(CSS.M("auto"));
      closePicker(c);
      expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
    });
  });

  describe("mixed", () => {
    it("should mark the swatch as mixed instead of auto", () => {
      const c = render(<Color.Swatch mixed fallback={RED} onChange={vi.fn()} />, {
        wrapper: Wrapper,
      });
      expect(swatchOf(c).className).toContain(CSS.M("mixed"));
      expect(swatchOf(c).className).not.toContain(CSS.M("auto"));
    });

    it("should call onChange with the picked color", () => {
      const onChange = vi.fn();
      const c = render(<Color.Swatch mixed fallback={RED} onChange={onChange} />, {
        wrapper: Wrapper,
      });
      fireEvent.click(swatchOf(c));
      pick(c, BLUE);
      expect(color.hex(onChange.mock.calls[0][0])).toEqual(`#${BLUE}`);
    });

    it("should call onChange with undefined when the user picks Auto", () => {
      const onChange = vi.fn();
      const c = render(<Color.Swatch mixed fallback={RED} onChange={onChange} />, {
        wrapper: Wrapper,
      });
      fireEvent.click(swatchOf(c));
      fireEvent.click(c.getByLabelText("Auto"));
      expect(onChange).toHaveBeenCalledExactlyOnceWith(undefined);
    });

    it("should show a pending pick in place of the stripes", () => {
      const c = render(
        <Color.Swatch mixed fallback={RED} onChange={vi.fn()} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, BLUE);
      expect(swatchOf(c).className).not.toContain(CSS.M("mixed"));
    });
  });

  describe("custom", () => {
    it("should show a color wheel in place of the value", () => {
      const c = render(<Color.Swatch custom value={RED} onChange={vi.fn()} />, {
        wrapper: Wrapper,
      });
      expect(c.getByLabelText("Custom color").className).toContain(CSS.M("custom"));
    });

    it("should not mark an absent value as auto", () => {
      const c = render(<Color.Swatch custom fallback={RED} onChange={vi.fn()} />, {
        wrapper: Wrapper,
      });
      expect(swatchOf(c).className).not.toContain(CSS.M("auto"));
    });
  });

  describe("onlyChangeOnBlur", () => {
    it("should not call onChange while the picker is open", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      pick(c, BLUE);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should show the pending color on the swatch while the picker is open", () => {
      const c = render(
        <Color.Swatch value={RED} onChange={vi.fn()} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      expect(swatchOf(c).style.getPropertyValue(SWATCH_VAR)).toEqual(
        color.cssString(`#${GREEN}`),
      );
    });

    it("should call onChange once with the last color when the picker closes", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      pick(c, BLUE);
      closePicker(c);
      expect(onChange).toHaveBeenCalledOnce();
      expect(color.hex(onChange.mock.calls[0][0])).toEqual(`#${BLUE}`);
    });

    it("should call onChange when a click on the swatch closes the picker", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      fireEvent.click(swatchOf(c));
      expect(onChange).toHaveBeenCalledOnce();
      expect(color.hex(onChange.mock.calls[0][0])).toEqual(`#${GREEN}`);
    });

    it("should not call onChange when the picker closes without a change", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      closePicker(c);
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should drop a pending color when the user picks a named color", () => {
      const onChange = vi.fn();
      const onNamedChange = vi.fn();
      const c = render(
        <Color.Swatch
          value={RED}
          onChange={onChange}
          onlyChangeOnBlur
          library={{
            palettes: [
              {
                key: "series",
                name: "Series",
                colors: [{ key: "ocean", name: "Ocean", light: "#3e8bff" }],
              },
            ],
            onNamedChange,
          }}
        />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      fireEvent.click(c.getByText("Library"));
      fireEvent.click(c.getByLabelText("Ocean"));
      closePicker(c);
      expect(onNamedChange).toHaveBeenCalledExactlyOnceWith("ocean");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should drop a pending change when the swatch unmounts", () => {
      const onChange = vi.fn();
      const c = render(
        <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      pick(c, GREEN);
      c.unmount();
      expect(onChange).not.toHaveBeenCalled();
    });

    describe("dismissed by a click outside", () => {
      it("should call onChange once with the last color", () => {
        const onChange = vi.fn();
        const c = render(
          <Color.Swatch value={RED} onChange={onChange} onlyChangeOnBlur />,
          { wrapper: Wrapper },
        );
        fireEvent.click(swatchOf(c));
        pick(c, GREEN);
        pick(c, BLUE);
        fireEvent.pointerDown(document.body, {
          clientX: window.innerWidth - 1,
          clientY: window.innerHeight - 1,
        });
        expect(onChange).toHaveBeenCalledOnce();
        expect(color.hex(onChange.mock.calls[0][0])).toEqual(`#${BLUE}`);
      });
    });
  });

  describe("library", () => {
    const PALETTES: Color.Palette[] = [
      {
        key: "series",
        name: "Series",
        colors: [{ key: "ocean", name: "Ocean", light: "#3e8bff" }],
      },
    ];

    it("should name the linked color in its tooltip", () => {
      vi.useFakeTimers();
      const delay = TimeSpan.milliseconds(100);
      const c = render(
        <Tooltip.Config delay={delay} skipDelay={delay}>
          <Color.Swatch
            value="#3e8bff"
            onChange={vi.fn()}
            library={{ palettes: PALETTES, named: "ocean", onNamedChange: vi.fn() }}
          />
        </Tooltip.Config>,
        { wrapper: Wrapper },
      );
      const swatch = swatchOf(c);
      swatch.getBoundingClientRect = mockBoundingClientRect(0, 0, 20, 20);
      fireEvent.pointerOver(swatch, { pointerType: "mouse" });
      act(() => {
        vi.advanceTimersByTime(200);
      });
      expect(document.querySelector(`.${CSS.B("tooltip")}`)?.textContent).toEqual(
        "Ocean. Click to change.",
      );
      vi.useRealTimers();
    });

    it("should show a pending edit as unlinked", () => {
      const c = render(
        <Color.Swatch
          value="#3e8bff"
          onChange={vi.fn()}
          onlyChangeOnBlur
          library={{ palettes: PALETTES, named: "ocean", onNamedChange: vi.fn() }}
        />,
        { wrapper: Wrapper },
      );
      fireEvent.click(swatchOf(c));
      expect(c.getByLabelText("Ocean").className).toContain(CSS.M("selected"));
      fireEvent.click(c.getByText("Custom"));
      pick(c, GREEN);
      fireEvent.click(c.getByText("Library"));
      expect(c.getByLabelText("Ocean").className).not.toContain(CSS.M("selected"));
    });
  });

  describe("disabled", () => {
    it("should not open the picker on a click", () => {
      const c = render(<Color.Swatch value={RED} onChange={vi.fn()} disabled />, {
        wrapper: Wrapper,
      });
      fireEvent.click(swatchOf(c));
      expect(c.queryByLabelText("Hex")).toBeNull();
    });
  });

  describe("drop", () => {
    const renderPair = (target: ReactElement): HTMLElement[] => {
      const c = render(
        <Haul.Provider>
          <Color.Swatch value={RED} onChange={vi.fn()} />
          {target}
        </Haul.Provider>,
        { wrapper: Wrapper },
      );
      const swatches = [
        ...c.container.querySelectorAll<HTMLElement>(`.${CSS.B("color-swatch")}`),
      ];
      fireEvent.dragStart(swatches[0], { dataTransfer: new DataTransfer() });
      return swatches;
    };

    it("should offer a changeable swatch as a drop target", () => {
      const [, target] = renderPair(
        <Color.Swatch value={`#${BLUE}`} onChange={vi.fn()} />,
      );
      expect(target.className).toContain(CSS.B("haul-drop-region"));
    });

    it("should not offer a swatch that cannot change as a drop target", () => {
      const [, target] = renderPair(<Color.Swatch value={`#${BLUE}`} />);
      expect(target.className).not.toContain(CSS.B("haul-drop-region"));
    });

    it("should not take a drop while disabled", () => {
      const onChange = vi.fn();
      const [, target] = renderPair(
        <Color.Swatch value={`#${BLUE}`} onChange={onChange} disabled />,
      );
      expect(target.className).not.toContain(CSS.B("haul-drop-region"));
      fireEvent.drop(target, { dataTransfer: new DataTransfer() });
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should call onChange with the dropped color", () => {
      const onChange = vi.fn();
      const [, target] = renderPair(
        <Color.Swatch value={`#${BLUE}`} onChange={onChange} />,
      );
      fireEvent.drop(target, { dataTransfer: new DataTransfer() });
      expect(onChange).toHaveBeenCalledWith(color.construct(RED));
    });

    describe("from another source", () => {
      const OTHER: Haul.Item = { type: "other", key: "other" };
      const COLOR: Haul.Item = { type: "color", key: RED };

      const Source = ({ items }: { items: Haul.Item[] }): ReactElement => {
        const { startDrag } = Haul.useDrag({ type: "source" });
        return (
          <div
            data-testid="source"
            onDragStart={(event) => startDrag({ event, items, preview: null })}
          />
        );
      };

      const renderSource = (items: Haul.Item[], onChange = vi.fn()): RenderResult => {
        const c = render(
          <Haul.Provider>
            <Source items={items} />
            <Color.Swatch value={`#${BLUE}`} onChange={onChange} />
          </Haul.Provider>,
          { wrapper: Wrapper },
        );
        fireEvent.dragStart(c.getByTestId("source"), {
          dataTransfer: new DataTransfer(),
        });
        return c;
      };

      it("should take the color from a drag that mixes in other items", () => {
        const onChange = vi.fn();
        const c = renderSource([OTHER, COLOR], onChange);
        fireEvent.drop(swatchOf(c), { dataTransfer: new DataTransfer() });
        expect(onChange).toHaveBeenCalledWith(color.construct(RED));
      });

      it("should reject a drag without a color", () => {
        const onChange = vi.fn();
        const c = renderSource([OTHER], onChange);
        fireEvent.drop(swatchOf(c), { dataTransfer: new DataTransfer() });
        expect(onChange).not.toHaveBeenCalled();
      });
    });
  });
});
