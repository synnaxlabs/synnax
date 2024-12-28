import { Breadcrumb } from "@/breadcrumb";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

describe("Breadcrumb", () => {
  describe("Breadcrumb", () => {
    it("should render a breadcrumb with a single segment", () => {
      const c = render(<Breadcrumb.Breadcrumb>{["Home"]}</Breadcrumb.Breadcrumb>);
      expect(c.getByText("Home")).toBeTruthy();
    });
    it("should render a breadcrumb with multiple segments", () => {
      const c = render(
        <Breadcrumb.Breadcrumb>
          {["Home", "Settings", "Profile"]}
        </Breadcrumb.Breadcrumb>,
      );
      expect(c.getByText("Home")).toBeTruthy();
      expect(c.getByText("Settings")).toBeTruthy();
      expect(c.getByText("Profile")).toBeTruthy();
      expect(c.getAllByLabelText("synnax-icon-caret-right")).toHaveLength(2);
    });
  });
  describe("URL", () => {
    it("should render a breadcrumb multiple segments", () => {
      const c = render(<Breadcrumb.URL url="home/settings/profile" />);
      expect(c.getByText("Home")).toBeTruthy();
      expect(c.getByText("Settings")).toBeTruthy();
      expect(c.getByText("Profile")).toBeTruthy();
      expect(c.getAllByLabelText("synnax-icon-caret-right")).toHaveLength(3);
      const home = c.getByText("Home");
      expect(home.getAttribute("href")).toBe("home");
      const settings = c.getByText("Settings");
      expect(settings.getAttribute("href")).toBe("home/settings");
      const profile = c.getByText("Profile");
      expect(profile.getAttribute("href")).toBe("home/settings/profile");
    });
  });
});
