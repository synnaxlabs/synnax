// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, map, observe, type status, zod } from "@synnaxlabs/x";
import { type z } from "zod";

export interface FieldState<V = unknown> {
  value: V;
  status: status.Crude;
  touched: boolean;
  required: boolean;
}

export interface RequiredGetOptions {
  optional?: false;
  defaultValue?: undefined;
}

export interface DefaultGetOptions<V> {
  optional?: boolean;
  defaultValue: V;
}

export interface OptionalGetOptions {
  optional: true;
  defaultValue?: undefined;
}

export interface ExtensionGetOptions<V> {
  optional?: boolean;
  defaultValue?: V;
}

export type GetOptions<V> =
  | RequiredGetOptions
  | OptionalGetOptions
  | DefaultGetOptions<V>
  | ExtensionGetOptions<V>;

const getVariant = (issue: z.core.$ZodIssue): status.Variant => {
  if (issue.code === "custom" && issue.params != null && "variant" in issue.params)
    return issue.params.variant;
  return "error";
};

export class State<Z extends z.ZodType> extends observe.Observer<void> {
  private readonly schema?: Z;
  values: z.infer<Z>;
  initialValues: z.infer<Z>;

  private readonly statuses: Map<string, status.Crude>;
  private readonly touched: Set<string>;
  private readonly cachedRefs: Map<string, {}>;

  constructor(values: z.infer<Z>, schema?: Z) {
    super();
    this.schema = schema;
    this.values = deep.copy(values);
    this.statuses = new Map();
    this.touched = new Set();
    this.cachedRefs = new Map();
    this.initialValues = deep.copy(this.values);
  }

  setValue(path: string, value: unknown) {
    if (path == "") this.values = deep.copy(value) as z.infer<Z>;
    else deep.set(this.values, path, value);
    this.checkTouched(path, value);
    this.updateCachedRefs(path);
  }

  checkTouched(path: string, value: unknown) {
    const initialValue = deep.get(this.initialValues, path, { optional: true });
    const equalsInitial = deep.equal(initialValue, value);
    if (equalsInitial) this.clearTouched(path);
    else this.setTouched(path);
  }

  setStatus(path: string, status: status.Crude) {
    this.statuses.set(path, status);
    this.updateCachedRefs(path);
  }

  clearStatus(path: string = "") {
    this.statuses.delete(path);
    this.updateCachedRefs(path);
  }

  reset(initialValues?: z.infer<Z>) {
    if (initialValues != null) this.initialValues = initialValues;
    const nextValues = deep.copy(this.initialValues);
    this.statuses.clear();
    this.touched.clear();
    const cachedRefsToClear = new Set<string>();
    this.cachedRefs.forEach((_, path) => {
      const prev = deep.get(this.values, path, { optional: true });
      const next = deep.get(nextValues, path, { optional: true });
      if (prev !== next) cachedRefsToClear.add(path);
    });
    cachedRefsToClear.forEach((path) => this.cachedRefs.delete(path));
    this.values = nextValues;
  }

  setCurrentStateAsInitialValues() {
    this.initialValues = deep.copy(this.values);
    this.touched.clear();
    this.cachedRefs.clear();
  }

  setTouched(path: string) {
    this.touched.add(path);
    this.updateCachedRefs(path);
  }

  clearTouched(path: string = "") {
    if (path === "") this.touched.clear();
    else this.touched.delete(path);
    this.updateCachedRefs(path);
    this.notify();
  }

  remove(path: string) {
    deep.remove(this.values, path);
    this.statuses.delete(path);
    this.clearTouched(path);
    this.cachedRefs.delete(path);
  }

  validate(path: string = "", validateChildren: boolean = true): boolean {
    if (this.schema == null) return true;
    const result = this.schema.safeParse(this.values);
    return this.processValidation(path, validateChildren, result);
  }

  async validateAsync(
    path: string = "",
    validateChildren: boolean = true,
  ): Promise<boolean> {
    if (this.schema == null) return true;
    const result = await this.schema.safeParseAsync(this.values);
    return this.processValidation(path, validateChildren, result);
  }

  private processValidation(
    path: string = "",
    validateChildren: boolean = true,
    result: z.ZodSafeParseResult<z.infer<Z>>,
  ): boolean {
    if (this.schema == null) return true;
    const cachedRefsToClear = new Set<string>();
    this.statuses.forEach((status, childPath) => {
      if (deep.pathsMatch(childPath, path) && status.variant !== "success")
        cachedRefsToClear.add(childPath);
    });
    cachedRefsToClear.forEach((path) => this.clearStatus(path));
    // Parse was a complete success. No errors encountered.
    if (result.success) return true;

    let success = true;
    let matcher = (a: string, b: string) => a === b;
    if (validateChildren) matcher = (a: string, b: string) => deep.pathsMatch(a, b);

    result.error.issues.forEach((issue) => {
      const { message } = issue;
      const issuePath = issue.path.join(".");
      // If we're only validating a sub-path and it doesn't match a particular issue,
      // skip it.
      if (!matcher(issuePath, path)) return;
      const variant = getVariant(issue);
      if (variant !== "warning") success = false;
      this.setStatus(issuePath, { key: issuePath, variant, message });
    });
    return success;
  }

  get hasBeenTouched() {
    return this.touched.size > 0;
  }

  getState<V>(path: string, opts?: RequiredGetOptions): FieldState<V>;
  getState<V>(path: string, opts?: DefaultGetOptions<V>): FieldState<V>;
  getState<V>(path: string, opts?: OptionalGetOptions): FieldState<V> | null;
  getState<V>(path: string, opts?: ExtensionGetOptions<V>): FieldState<V> | null;

  getState<V>(path: string, opts: GetOptions<V> = {}): FieldState<V> | null {
    const { optional = false, defaultValue = undefined } = opts;
    const cachedRef = map.getOrSetDefault(this.cachedRefs, path, {}) as FieldState<V>;
    let value = deep.get<V, z.infer<Z>>(this.values, path, {
      optional: optional || defaultValue != null,
    });
    if (value == null) {
      if (defaultValue == null) return null;
      value = defaultValue;
      this.setValue(path, value);
    }
    cachedRef.value = value;
    cachedRef.required = false;
    if (this.schema != null) {
      const fieldSchema = zod.getFieldSchema(this.schema, path, { optional: true });
      if (fieldSchema != null)
        cachedRef.required = !fieldSchema.safeParse(undefined).success;
    }
    cachedRef.status = map.getOrSetDefault(this.statuses, path, {
      key: path,
      variant: "success",
      message: "",
    });
    cachedRef.touched = this.touched.has(path);
    return cachedRef;
  }

  private updateCachedRefs(fieldPath: string) {
    this.cachedRefs.forEach((_, refPath) => {
      if (deep.pathsMatch(refPath, fieldPath) || deep.pathsMatch(fieldPath, refPath))
        this.cachedRefs.set(refPath, {});
    });
  }
}
