// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// ResolveWriteConfig moves the staged channel and fields of each endpoint of cfg into
// a new ICD named name, with one JSON message per endpoint, named after its method and
// path. It sets the ICD and message of each endpoint and clears cfg.Inline. A channel
// that snap does not hold binds no channel, but the data type of a new one.
// ResolveWriteConfig returns false and no ICD when cfg stages nothing. note receives
// the path of a field in cfg and each change to what the field sends.
func ResolveWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels icdv0.Channels,
	name string,
	cfg *WriteConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	staged := cfg.Inline.Endpoints
	found, err := channels.Of(
		ctx,
		snap,
		lo.Map(staged, func(e InlineWriteEndpoint, _ int) icdv0.InlineField {
			return icdv0.InlineField{Channel: e.Channel.Channel}
		}),
	)
	if err != nil {
		return icd.ICD{}, false, err
	}
	ic := icd.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icd.Message, len(staged)),
	}
	names := make(set.Set[string])
	for i, e := range staged {
		at := slices.IndexFunc(cfg.Endpoints, func(ep WriteEndpoint) bool {
			return ep.Key == e.Key
		})
		if at < 0 {
			return icd.ICD{}, false, errors.Newf(
				"staged endpoint %s is not an endpoint of the config",
				e.Key,
			)
		}
		ep := &cfg.Endpoints[at]
		request := strings.TrimSpace(string(ep.Method) + " " + ep.Path)
		msg, write, err := icd.LiftWrite(
			ctx,
			inlineWrite(e, request, "endpoints."+strconv.Itoa(at), note),
			found,
			note,
		)
		if err != nil {
			return icd.ICD{}, false, err
		}
		msg.Name = icdv0.UniqueName(names, request)
		ic.Messages[i] = msg
		ep.ICD, ep.Message = new(ic.Key), &write
	}
	cfg.Inline = nil
	return ic, true, nil
}

// inlineWrite returns the write of e, which sends request, for the lift. path is the
// path of e's endpoint in its config. note receives each field that the lift drops.
func inlineWrite(
	e InlineWriteEndpoint,
	request string,
	path string,
	note func(path, change string),
) icd.InlineWrite {
	fields := make([]icd.InlineWriteField, 0, len(e.Fields))
	for j, f := range e.Fields {
		fieldPath := path + ".fields." + strconv.Itoa(j)
		switch v := f.Variant.(type) {
		case StaticWriteField:
			fields = append(fields, icd.StaticInlineField{
				Path:    fieldPath,
				Pointer: v.Pointer,
				Value:   v.Value,
			})
		case GeneratedWriteField:
			fields = append(fields, icd.GeneratedInlineField{
				Path:       fieldPath,
				Pointer:    v.Pointer,
				Generator:  v.Generator,
				TimeFormat: v.TimeFormat,
			})
		case nil:
			note(fieldPath, "a null field was dropped")
		default:
			panic(fmt.Sprintf("unknown write field variant %T", f.Variant))
		}
	}
	c := e.Channel
	return icd.InlineWrite{
		Target: request,
		Channel: icd.InlineChannelField{
			Path:       path + ".channel",
			Pointer:    c.Pointer,
			JSONType:   c.JSONType,
			Channel:    c.Channel,
			Name:       c.Name,
			DataType:   c.DataType,
			TimeFormat: c.TimeFormat,
			EnumValues: c.EnumValues,
		},
		Fields: fields,
	}
}
