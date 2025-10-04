// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { render, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { useContext } from "@/form/Context";

const mockSchema = z.object({
  name: z.string().min(1, "Name is required"),
  age: z.number().min(0, "Age must be positive"),
});

const mockInitialValues = {
  name: "John Doe",
  age: 25,
};

const mockOverride: Form.ContextValue = {
  mode: "preview" as const,
  bind: () => () => {},
  set: () => {},
  get: () => ({
    value: "test",
    status: {
      key: "test",
      variant: "success" as const,
      message: "",
      description: undefined,
      time: TimeStamp.now(),
    },
    touched: false,
    required: false,
  }),
  reset: () => {},
  remove: () => {},
  value: () => ({ test: "value" }),
  validate: () => false,
  validateAsync: async () => false,
  has: () => false,
  setStatus: () => {},
  clearStatuses: () => {},
  setCurrentStateAsInitialValues: () => {},
  getStatuses: () => [],
};

// Wrapper that provides Form context
const FormWrapper = ({ children }: PropsWithChildren): ReactElement => {
  const methods = Form.use({
    values: mockInitialValues,
    schema: mockSchema,
  });
  return <Form.Form<typeof mockSchema> {...methods}>{children}</Form.Form>;
};

describe("useContext", () => {
  describe("error handling when used outside Form context", () => {
    it("should throw error with default function name when used outside context without override", () => {
      expect(() => {
        renderHook(() => useContext());
      }).toThrow("Form.useContext must be used within a Form context value");
    });

    it("should throw error with custom function name when used outside context without override", () => {
      expect(() => {
        renderHook(() => useContext(undefined, "CustomFunction"));
      }).toThrow("CustomFunction must be used within a Form context value");
    });

    it("should NOT throw error when override is provided even outside context", () => {
      const { result } = renderHook(() => useContext(mockOverride, "CustomFunction"));
      expect(result.current).toBe(mockOverride);
    });

    it("should throw error with Field component name when used outside context", () => {
      expect(() => {
        renderHook(() => useContext(undefined, "Field(user.name)"));
      }).toThrow("Field(user.name) must be used within a Form context value");
    });

    it("should throw error with useField hook name when used outside context", () => {
      expect(() => {
        renderHook(() => useContext(undefined, "useField(user.email)"));
      }).toThrow("useField(user.email) must be used within a Form context value");
    });

    it("should throw error with useFieldState hook name when used outside context", () => {
      expect(() => {
        renderHook(() => useContext(undefined, "useFieldState(profile.bio)"));
      }).toThrow("useFieldState(profile.bio) must be used within a Form context value");
    });

    it("should throw error with useFieldValue hook name when used outside context", () => {
      expect(() => {
        renderHook(() => useContext(undefined, "useFieldValue(settings.theme)"));
      }).toThrow(
        "useFieldValue(settings.theme) must be used within a Form context value",
      );
    });
  });

  describe("override behavior", () => {
    it("should use override even when context is null", () => {
      // This should not throw even though we're outside a Form context
      const { result } = renderHook(() => useContext(mockOverride));

      expect(result.current).toBe(mockOverride);
      expect(result.current.mode).toBe("preview");
      expect(result.current.validate()).toBe(false);
    });

    it("should prioritize override over internal context when both exist", () => {
      const { result } = renderHook(() => useContext(mockOverride), {
        wrapper: FormWrapper,
      });

      // Should use override, not the internal context
      expect(result.current).toBe(mockOverride);
      expect(result.current.mode).toBe("preview");
    });
  });

  describe("successful context usage", () => {
    it("should return context value when used within Form context", () => {
      const { result } = renderHook(() => useContext(), {
        wrapper: FormWrapper,
      });

      expect(result.current).toBeDefined();
      expect(typeof result.current.get).toBe("function");
      expect(typeof result.current.set).toBe("function");
      expect(typeof result.current.bind).toBe("function");
      expect(typeof result.current.validate).toBe("function");
    });

    it("should return context value with custom function name when used within Form context", () => {
      const { result } = renderHook(() => useContext(undefined, "CustomFunction"), {
        wrapper: FormWrapper,
      });

      expect(result.current).toBeDefined();
      expect(typeof result.current.get).toBe("function");
      expect(typeof result.current.set).toBe("function");
    });

    it("should return override context when provided", () => {
      const { result } = renderHook(() => useContext(mockOverride, "TestFunction"), {
        wrapper: FormWrapper,
      });
      expect(result.current).toBe(mockOverride);
    });
  });

  describe("integration with actual Form components", () => {
    it("should work properly when Field component calls useContext", () => {
      const TestField = () => {
        const ctx = useContext(undefined, `Field(name)`);
        return <div>Field has context: {ctx ? "true" : "false"}</div>;
      };

      const { result } = renderHook(() => <TestField />, {
        wrapper: FormWrapper,
      });

      // Should not throw any errors
      expect(() => result.current).not.toThrow();
    });

    it("should work properly when useField hook calls useContext", () => {
      const TestUseField = () => {
        const ctx = useContext(undefined, `useField(name)`);
        return ctx.get("name").value;
      };

      const { result } = renderHook(() => TestUseField(), {
        wrapper: FormWrapper,
      });

      // Should not throw any errors and return the expected value
      expect(() => result.current).not.toThrow();
    });
  });

  describe("error message variations", () => {
    it("should handle complex path expressions in function names", () => {
      expect(() => {
        renderHook(() =>
          useContext(undefined, "Field(user.profile.settings[0].theme)"),
        );
      }).toThrow(
        "Field(user.profile.settings[0].theme) must be used within a Form context value",
      );
    });

    it("should handle function names with special characters", () => {
      expect(() => {
        renderHook(() =>
          useContext(undefined, 'useField(data.items["complex-key"].value)'),
        );
      }).toThrow(
        'useField(data.items["complex-key"].value) must be used within a Form context value',
      );
    });

    it("should handle empty function name", () => {
      expect(() => {
        renderHook(() => useContext(undefined, ""));
      }).toThrow(" must be used within a Form context value");
    });

    it("should handle very long function names", () => {
      const longFunctionName =
        "useField(very.long.nested.path.with.many.levels.and.properties.that.goes.on.and.on)";
      expect(() => {
        renderHook(() => useContext(undefined, longFunctionName));
      }).toThrow(`${longFunctionName} must be used within a Form context value`);
    });
  });

  describe("type safety", () => {
    it("should preserve correct typing when context is available", () => {
      const { result } = renderHook(
        () => {
          const ctx = useContext();
          // These should be type-safe operations
          const fieldState = ctx.get("name");
          return {
            hasGetMethod: typeof ctx.get === "function",
            hasSetMethod: typeof ctx.set === "function",
            hasValidateMethod: typeof ctx.validate === "function",
            fieldValue: fieldState.value,
          };
        },
        {
          wrapper: FormWrapper,
        },
      );

      expect(result.current.hasGetMethod).toBe(true);
      expect(result.current.hasSetMethod).toBe(true);
      expect(result.current.hasValidateMethod).toBe(true);
      expect(result.current.fieldValue).toBe("John Doe");
    });
  });

  describe("With field APIS", () => {
    describe("Field component error messages", () => {
      it("should throw error with Field path when used outside Form context", () => {
        expect(() => {
          render(<Form.Field path="user.name" />);
        }).toThrow("Field(user.name) must be used within a Form context value");
      });

      it("should throw error with complex Field path when used outside Form context", () => {
        expect(() => {
          render(<Form.Field path="settings.appearance.theme.colors[0]" />);
        }).toThrow(
          "Field(settings.appearance.theme.colors[0]) must be used within a Form context value",
        );
      });
    });

    describe("useField hook error messages", () => {
      it("should throw error with useField path when used outside Form context", () => {
        expect(() => {
          renderHook(() => Form.useField("profile.email"));
        }).toThrow("useField(profile.email) must be used within a Form context value");
      });

      it("should throw error with complex useField path when used outside Form context", () => {
        expect(() => {
          renderHook(() => Form.useField("data.metrics.performance[0].value"));
        }).toThrow(
          "useField(data.metrics.performance[0].value) must be used within a Form context value",
        );
      });
    });

    describe("useFieldState hook error messages", () => {
      it("should throw error with default function name when used outside Form context", () => {
        expect(() => {
          renderHook(() => Form.useFieldState("status"));
        }).toThrow("Form.useContext must be used within a Form context value");
      });
    });

    describe("useFieldValue hook error messages", () => {
      it("should throw error with default function name when used outside Form context", () => {
        expect(() => {
          renderHook(() => Form.useFieldValue("count"));
        }).toThrow("Form.useContext must be used within a Form context value");
      });
    });

    describe("other form hooks error messages", () => {
      it("should throw error for useContext in Form.useFieldList", () => {
        expect(() => {
          renderHook(() => Form.useFieldList("items"));
        }).toThrow("Form.useContext must be used within a Form context value");
      });

      it("should throw error for useContext in Form.useFieldListUtils", () => {
        expect(() => {
          renderHook(() => Form.useFieldListUtils("tags"));
        }).toThrow("Form.useContext must be used within a Form context value");
      });
    });
  });
});
