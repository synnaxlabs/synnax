// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action } from "@/access/action/types.gen";
import { allowRequest } from "@/access/enforce";
import { type policy } from "@/access/policy";
import { type query } from "@/query";
import { type resource } from "@/resource";
import { user } from "@/user";

/** A permission question: may the subject apply the action to the objects? */
export interface GrantedQuery {
  objects: resource.ID | resource.ID[];
  action: Action;
}

/** The user a question asks about. Its role selects the policies that answer it. */
export interface Grantee extends Pick<user.User, "key" | "role"> {}

interface Verdict {
  role: string;
  policies: policy.Policy[];
  verdict: boolean;
}

export interface GrantedConfig {
  policies: policy.Client;
}

/**
 * The cached read surface for permission checks, answering {@link GrantedQuery}
 * questions against the cached policies of the grantee's role. Snapshot reads memoize
 * the verdict per query object, so render-hot callers pay a policy evaluation only when
 * the policies actually change.
 */
export class Granted {
  private readonly policies: policy.Client;
  private readonly queries = new Map<string, policy.RetrieveMultipleParams>();
  private readonly verdicts = new WeakMap<GrantedQuery, Verdict>();

  constructor(cfg: GrantedConfig) {
    this.policies = cfg.policies;
  }

  // A stable query per role keeps repeat reads on the retriever's memo.
  private queryFor({ role }: Grantee): policy.RetrieveMultipleParams {
    let held = this.queries.get(role);
    if (held == null) {
      held = { roles: [role] };
      this.queries.set(role, held);
    }
    return held;
  }

  private evaluate(
    grantee: Grantee,
    q: GrantedQuery,
    policies: policy.Policy[],
  ): boolean {
    const subject: resource.ID = user.resourceID(grantee.key);
    return allowRequest({ subject, objects: q.objects, action: q.action }, policies);
  }

  /** Fetches the policies of the grantee's role and answers the question. */
  async retrieve(grantee: Grantee, q: GrantedQuery): Promise<boolean> {
    return await this.watch(grantee, q).retrieve();
  }

  /**
   * Returns the live verdict. Its handlers fire only when the verdict flips: policy
   * churn that cannot change the answer never reaches them. Repeat cached reads with
   * the same query object skip the policy evaluation until the cached policies change.
   */
  watch(grantee: Grantee, q: GrantedQuery): query.Live<boolean> {
    const policies = this.policies.watch(this.queryFor(grantee));
    const verdictOf = (window: query.Window<policy.Policy, unknown> | undefined) =>
      window == null ? undefined : this.verdict(grantee, q, window.records);
    return {
      retrieve: async (options) =>
        this.verdict(grantee, q, (await policies.retrieve(options)).records),
      getCached: () => verdictOf(policies.getCached()),
      onChange: (handler) => {
        let prev = verdictOf(policies.getCached());
        return policies.onChange((window) => {
          const next = verdictOf(window);
          if (next == null || next === prev) return;
          prev = next;
          handler(next);
        });
      },
    };
  }

  private verdict(
    grantee: Grantee,
    q: GrantedQuery,
    policies: policy.Policy[],
  ): boolean {
    const { role } = grantee;
    const held = this.verdicts.get(q);
    if (held != null && held.policies === policies && held.role === role)
      return held.verdict;
    const verdict = this.evaluate(grantee, q, policies);
    this.verdicts.set(q, { role, policies, verdict });
    return verdict;
  }
}
