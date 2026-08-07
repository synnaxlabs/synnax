// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v1 "github.com/synnaxlabs/arc/ir/versions/v1"

// NodeMember builds a leaf Member referencing the node with the given key.
func NodeMember(key string) Member { return v1.NodeMember(key) }

// ScopeMember builds a Member wrapping the given nested Scope.
func ScopeMember(s Scope) Member { return v1.ScopeMember(s) }
