// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// textSitePrefix starts the key of each staged text site. It does not start with
// renamePrefix, so StagedRenames reads no site.
const textSitePrefix = "sy_v63_staged_text_site/"

// TextSite is a reference in stored text that the upgrade left on the old name of a
// renamed channel.
type TextSite struct {
	// Holder is the resource whose text holds the reference: a calculated channel or an
	// Arc.
	Holder ID `json:"holder"`
	// Function is the key of the graph function whose body holds the reference, or
	// empty when the text is not the body of a graph function.
	Function string `json:"function,omitempty"`
	// Name is the old name, and NewName the new name of the channel.
	Name    string `json:"name"`
	NewName string `json:"new_name"`
	// Line counts from 1, and Column counts characters from 0.
	Line   int `json:"line"`
	Column int `json:"column"`
	// Captured is true when a symbol that the text declares takes the old or the new
	// name at the reference. Folder is the first folder on the path of the channel that
	// Arc cannot read bare yet, or empty. Both are unset when the text does not parse
	// or bind.
	Captured bool   `json:"captured"`
	Folder   string `json:"folder,omitempty"`
}

func textSiteKey(s TextSite) []byte {
	return fmt.Appendf(
		nil,
		"%s%s/%s/%d:%d/%s",
		textSitePrefix,
		s.Holder,
		s.Function,
		s.Line,
		s.Column,
		s.Name,
	)
}

// StageTextSite records site for the report of the upgrade.
func StageTextSite(ctx context.Context, tx kv.ReadWriter, site TextSite) error {
	value, err := json.Marshal(site)
	if err != nil {
		return err
	}
	return tx.Set(ctx, textSiteKey(site), value)
}

// StagedTextSites returns each site that StageTextSite recorded, in key order.
func StagedTextSites(r kv.Reader) (sites []TextSite, err error) {
	iter, err := r.OpenIterator(kv.IterPrefix([]byte(textSitePrefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		var s TextSite
		if err = json.Unmarshal(iter.Value(), &s); err != nil {
			return nil, errors.Wrapf(
				err,
				"staged text site %q",
				bytes.TrimPrefix(iter.Key(), []byte(textSitePrefix)),
			)
		}
		sites = append(sites, s)
	}
	return sites, iter.Error()
}

// ClearTextSites deletes each staged text site.
func ClearTextSites(ctx context.Context, tx kv.ReadWriter) error {
	sites, err := StagedTextSites(tx)
	if err != nil {
		return err
	}
	for _, s := range sites {
		if err = tx.Delete(ctx, textSiteKey(s)); err != nil {
			return err
		}
	}
	return nil
}
