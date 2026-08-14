// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { type Action } from "@/access/action/types.gen";
import { allowRequest } from "@/access/enforce";
import { type policy } from "@/access/policy";
import { ontology } from "@/ontology";
import { query } from "@/query";

/** A permission question: may the subject apply the action to the objects? */
export interface GrantedQuery {
  objects: ontology.ID | ontology.ID[];
  action: Action;
}

interface Verdict {
  subject: string;
  policies: policy.Policy[];
  verdict: boolean;
}

interface SubjectEntry {
  key: string;
  query: { for: ontology.ID };
}

export interface GrantedConfig {
  policies: policy.Client;
}

/**
 * The cached read surface for permission checks, answering {@link GrantedQuery}
 * questions against the subject's cached policies. Snapshot reads memoize the
 * verdict per query object, so render-hot callers pay a policy evaluation only
 * when the policies actually change.
 */
export class Granted {
  private readonly policies: policy.Client;
  private readonly subjects = new Map<string, SubjectEntry>();
  private readonly verdicts = new WeakMap<GrantedQuery, Verdict>();

  constructor(cfg: GrantedConfig) {
    this.policies = cfg.policies;
  }

  // A subject's entry holds its canonical key and a stable `{ for }` query, so
  // repeat reads stringify once and stay on the retriever's normalization memo.
  private subjectFor(subject: ontology.ID): SubjectEntry {
    const key = ontology.idToString(subject);
    let held = this.subjects.get(key);
    if (held == null) {
      held = { key, query: { for: subject } };
      this.subjects.set(key, held);
    }
    return held;
  }

  /** Fetches the subject's policies and answers the question. */
  async retrieve(subject: ontology.ID, q: GrantedQuery): Promise<boolean> {
    const policies = await this.policies.retrieveForSubject(subject);
    return allowRequest({ subject, objects: q.objects, action: q.action }, policies);
  }

  /**
   * Subscribes to the verdict, firing only when it flips: policy churn that
   * cannot change the answer never reaches the handler.
   */
  onChange(
    subject: ontology.ID,
    q: GrantedQuery,
    handler: (granted: boolean) => void,
  ): destructor.Destructor {
    const { query: policiesQuery } = this.subjectFor(subject);
    const evaluate = (policies: policy.Policy[]): boolean =>
      allowRequest({ subject, objects: q.objects, action: q.action }, policies);
    const cached = this.policies.getCached(policiesQuery);
    let prev = query.isLive(cached) ? evaluate(cached) : undefined;
    return this.policies.onChange(policiesQuery, (result) => {
      if (!query.isLive(result)) return;
      const next = evaluate(result);
      if (next === prev) return;
      prev = next;
      handler(next);
    });
  }

  /**
   * Returns the cached verdict, or undefined when the subject's policies are
   * not cached. Repeat calls with the same query object skip the policy
   * evaluation until the cached policies change.
   */
  getCached(subject: ontology.ID, q: GrantedQuery): boolean | undefined {
    const { key: subjectKey, query: policiesQuery } = this.subjectFor(subject);
    const cached = this.policies.getCached(policiesQuery);
    if (!query.isLive(cached)) return undefined;
    const held = this.verdicts.get(q);
    if (held != null && held.policies === cached && held.subject === subjectKey)
      return held.verdict;
    const verdict = allowRequest(
      { subject, objects: q.objects, action: q.action },
      cached,
    );
    this.verdicts.set(q, { subject: subjectKey, policies: cached, verdict });
    return verdict;
  }
}
