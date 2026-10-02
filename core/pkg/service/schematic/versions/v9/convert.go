// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9

import (
	"cmp"
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// opaqueConfigFields names element config fields whose contents carry semantic keys
// (telem pipeline segment names) and must not be case-converted.
var opaqueConfigFields = set.New("props")

// NormalizeConfigKeys converts a config payload's field keys from the camelCase the
// Console writes verbatim to the snake_case wire form of the element config union,
// along with the variant discriminator value, whose legacy form matched the camelCase
// symbol registry keys. Already snake_case input passes through unchanged, so the
// conversion is idempotent. Values under opaque fields are left untouched.
func NormalizeConfigKeys(raw msgpack.EncodedJSON) msgpack.EncodedJSON {
	if raw == nil {
		return nil
	}
	out := normalizeConfigMap(raw)
	if variant, ok := out["variant"].(string); ok {
		out["variant"] = camelToSnakeKey(variant)
	}
	return out
}

func normalizeConfigMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, val := range m {
		nk := camelToSnakeKey(k)
		if opaqueConfigFields.Contains(nk) {
			out[nk] = val
			continue
		}
		out[nk] = normalizeConfigValue(val)
	}
	return out
}

func normalizeConfigValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return normalizeConfigMap(t)
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = normalizeConfigValue(val)
		}
		return out
	default:
		return v
	}
}

// camelToSnakeKey converts a camelCase identifier to snake_case, leaving already
// snake_case identifiers unchanged.
func camelToSnakeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// DecodeElementConfig validates an opaque config payload against the element config
// union. It wraps validate.ErrValidation when the payload names no variant, names one
// the union does not, or carries fields that do not fit the variant it names.
func DecodeElementConfig(raw msgpack.EncodedJSON) (ElementConfig, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return ElementConfig{}, err
	}
	var cfg ElementConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ElementConfig{}, errors.Wrapf(
			validate.ErrValidation, "invalid element config: %s", err,
		)
	}
	// A null payload decodes to a nil variant without erroring, which would persist an
	// entry no client can read back.
	if cfg.Variant == nil {
		return ElementConfig{}, errors.Wrap(
			validate.ErrValidation, "element config names no variant",
		)
	}
	return cfg, nil
}

// ElementConfigFields returns a config's wire fields as an opaque map for partial
// merging.
func ElementConfigFields(cfg ElementConfig) (msgpack.EncodedJSON, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var m msgpack.EncodedJSON
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// extractTelemArgs rewrites a config's stored telem pipeline specs into the semantic
// arguments the schema now declares, in place on the normalized wire map. Channel keys
// are read from the pipeline's well-known segments; a zero channel (the legacy unset
// default) produces no argument. Spec fields are removed regardless so the entry
// decodes under the args-based schema.
func extractTelemArgs(cfg map[string]any) {
	variant, _ := cfg["variant"].(string)
	switch variant {
	case "value", "gauge":
		if ch, ok := segProp(cfg["telem"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		if w, ok := segProp(cfg["telem"], "rollingAverage", "windowSize"); ok {
			cfg["rolling_average"] = floorWindow(w)
		}
		if p, ok := segProp(cfg["telem"], "stringifier", "precision"); ok {
			cfg["precision"] = p
		}
		if n, ok := segProp(cfg["telem"], "stringifier", "notation"); ok {
			cfg["notation"] = n
		}
		delete(cfg, "telem")
		delete(cfg, "background_telem")
	case "string_display":
		if ch, ok := segProp(cfg["telem"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		delete(cfg, "telem")
	case "light":
		if ch, ok := segProp(cfg["source"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		if b, ok := segProp(cfg["source"], "threshold", "trueBound"); ok {
			cfg["threshold"] = b
		}
		delete(cfg, "source")
	case "state_indicator":
		if ch, ok := segProp(cfg["source"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		delete(cfg, "source")
	case "setpoint":
		if ch, ok := segProp(cfg["sink"], "setter", "channel"); ok {
			cfg["command_channel"] = ch
		}
		delete(cfg, "source")
		delete(cfg, "sink")
	case "scale", "tank":
		indicator, _ := cfg[legacyIndicators[variant]].(map[string]any)
		if ch, ok := segProp(indicator["telem"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		if w, ok := segProp(indicator["telem"], "rollingAverage", "windowSize"); ok {
			cfg["rolling_average"] = floorWindow(w)
		}
		delete(indicator, "telem")
	case "button", "select", "input":
		if ch, ok := segProp(cfg["sink"], "setter", "channel"); ok {
			cfg["command_channel"] = ch
		}
		delete(cfg, "sink")
	default:
		if _, ok := cfg["source"]; ok {
			if ch, k := segProp(cfg["source"], "valueStream", "channel"); k {
				cfg["state_channel"] = ch
			}
			delete(cfg, "source")
		}
		if _, ok := cfg["sink"]; ok {
			if ch, k := segProp(cfg["sink"], "setter", "channel"); k {
				cfg["command_channel"] = ch
			}
			delete(cfg, "sink")
		}
	}
	if ctl, ok := cfg["control"].(map[string]any); ok {
		if chip, ok := ctl["chip"].(map[string]any); ok {
			if sink, ok := chip["sink"].(map[string]any); ok {
				if props, ok := sink["props"].(map[string]any); ok {
					if a, ok := props["authority"]; ok {
						ctl["authority"] = a
					}
				}
			}
		}
		delete(ctl, "chip")
		delete(ctl, "indicator")
	}
}

// floorWindow rounds a stored averaging window down to a whole sample count. The
// Console's numeric input stored a typed fraction unrounded, and its window held the
// floor of that many samples.
func floorWindow(v any) any {
	switch t := v.(type) {
	case float64:
		return int64(math.Floor(t))
	case float32:
		return int64(math.Floor(float64(t)))
	}
	return v
}

// segProp reads a property from a named segment of a stored pipeline spec, reporting
// false when any layer is missing or the value is a zero channel.
func segProp(spec any, segment, prop string) (any, bool) {
	m, ok := spec.(map[string]any)
	if !ok {
		return nil, false
	}
	props, ok := m["props"].(map[string]any)
	if !ok {
		return nil, false
	}
	segments, ok := props["segments"].(map[string]any)
	if !ok {
		// Single-segment pipelines store the spec at the top level.
		segments = map[string]any{segment: m}
	}
	seg, ok := segments[segment].(map[string]any)
	if !ok {
		return nil, false
	}
	segProps, ok := seg["props"].(map[string]any)
	if !ok {
		return nil, false
	}
	v, ok := segProps[prop]
	if !ok {
		return nil, false
	}
	if prop == "channel" && isZeroNumber(v) {
		return nil, false
	}
	return v, true
}

// isZeroNumber reports whether v is a numeric zero of any width, since msgpack decodes
// a stored integer to the narrowest type that holds it.
func isZeroNumber(v any) bool {
	rv := reflect.ValueOf(v)
	switch {
	case rv.CanInt():
		return rv.Int() == 0
	case rv.CanUint():
		return rv.Uint() == 0
	case rv.CanFloat():
		return rv.Float() == 0
	}
	return false
}

// normalizePage lifts an off-page reference's legacy page, a bare schematic key, into
// the typed page reference. An empty key meant no page.
func normalizePage(cfg map[string]any) {
	if cfg["variant"] != "off_page_reference" {
		return
	}
	key, ok := cfg["page"].(string)
	if !ok {
		return
	}
	if key == "" {
		delete(cfg, "page")
		return
	}
	cfg["page"] = map[string]any{"type": "schematic", "key": key}
}

// legacyStop is a gradient stop of a legacy redline, positioned in [0, 1] across the
// redline's bounds.
type legacyStop struct {
	Key      string  `json:"key"`
	Color    any     `json:"color"`
	Position float64 `json:"position"`
}

// legacyRedline is the redline Consoles before v9 stored.
type legacyRedline struct {
	Bounds *struct {
		Lower float64 `json:"lower"`
		Upper float64 `json:"upper"`
	} `json:"bounds"`
	Gradient []legacyStop `json:"gradient"`
}

// bandRedline rewrites a value config's legacy redline into threshold bands, in place
// on the normalized wire map. Each stop becomes a band at its position scaled across
// the bounds. The bands interpolate and the lowest stop's color becomes the background,
// so every value keeps the fill the legacy renderer painted.
func bandRedline(cfg map[string]any) error {
	if cfg["variant"] != "value" {
		return nil
	}
	raw, ok := cfg["redline"]
	if !ok {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	var old legacyRedline
	if err = json.Unmarshal(b, &old); err != nil {
		return errors.Wrapf(validate.ErrValidation, "invalid redline: %s", err)
	}
	lower, upper := 0.0, 1.0
	if b := old.Bounds; b != nil {
		// The legacy renderer swapped reversed bounds before it scaled.
		lower, upper = min(b.Lower, b.Upper), max(b.Lower, b.Upper)
	}
	bands := make([]any, len(old.Gradient))
	for i, stop := range old.Gradient {
		bands[i] = map[string]any{
			"key":       stop.Key,
			"threshold": lower + stop.Position*(upper-lower),
			"color":     stop.Color,
		}
	}
	redline := map[string]any{"bands": bands}
	if len(old.Gradient) > 0 {
		redline["smooth"] = true
		lowest := slices.MinFunc(old.Gradient, func(a, b legacyStop) int {
			return cmp.Compare(a.Position, b.Position)
		})
		if !isZeroColor(lowest.Color) {
			cfg["background_color"] = lowest.Color
		}
	}
	cfg["redline"] = redline
	return nil
}

// zeroColorOpaqueFields names fields whose colors keep a zero value, alongside the
// fields already excluded from normalization. Band colors are required. Released
// Consoles painted a zero region color under state_overrides as transparent.
var zeroColorOpaqueFields = set.New("bands", "state_overrides")

// polygonZeroColorOpaqueFields adds the polygon's fill, which released Consoles also
// painted as transparent when zero.
var polygonZeroColorOpaqueFields = set.New("bands", "state_overrides", "fill_color")

// stripZeroColors deletes every color-valued field of cfg holding the zero color.
// Consoles before v9 stored transparent black for an unchosen color; v9 stores
// nothing, so the theme picks the color instead. Fields a release painted as
// transparent keep their zero.
func stripZeroColors(cfg map[string]any) {
	kept := zeroColorOpaqueFields
	if cfg["variant"] == "polygon" {
		kept = polygonZeroColorOpaqueFields
	}
	stripZeroColorsExcept(cfg, kept)
}

func stripZeroColorsExcept(v any, kept set.Set[string]) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if opaqueConfigFields.Contains(k) || kept.Contains(k) {
				continue
			}
			if strings.HasSuffix(k, "color") {
				if isZeroColor(val) {
					delete(t, k)
				}
				continue
			}
			stripZeroColorsExcept(val, kept)
		}
	case []any:
		for _, item := range t {
			stripZeroColorsExcept(item, kept)
		}
	}
}

// isZeroColor reports whether v decodes as the zero color in any stored encoding.
func isZeroColor(v any) bool {
	b, err := json.Marshal(v)
	if err != nil {
		return false
	}
	var c color.Color
	if err := json.Unmarshal(b, &c); err != nil {
		return false
	}
	return c == color.Color{}
}

// strokeAndFill renames the fields of a symbol with an outline and a body.
var strokeAndFill = map[string][]string{
	"color":            {"stroke_color"},
	"background_color": {"fill_color"},
}

// colorRenames maps each variant's legacy color fields to the fields named for the part
// they paint. A variant absent here keeps a legacy color as its stroke.
var colorRenames = map[string]map[string][]string{
	"box":                strokeAndFill,
	"circle":             strokeAndFill,
	"polygon":            strokeAndFill,
	"cylinder":           strokeAndFill,
	"tank":               strokeAndFill,
	"value":              strokeAndFill,
	"button":             {"color": {"fill_color"}},
	"input":              {"color": {"fill_color"}},
	"setpoint":           {"color": {"fill_color"}},
	"select":             {"color": {"fill_color"}},
	"off_page_reference": {"color": {"fill_color"}},
	"text_box":           {"color": {"text_color"}},
	"light":              {"color": {"stroke_color", "on_color"}},
	"scale":              {"color": {"level_color"}},
	"custom_actuator":    {"color": nil},
	"custom_static":      {"color": nil},
}

// legacyIndicators names the field each variant nested its level indicator under.
var legacyIndicators = map[string]string{"scale": "indicator", "tank": "fill"}

// indicatorRenames maps a legacy nested indicator's fields to the fields of the symbol
// that now extends the indicator. The indicator's own color is dropped: a scale painted
// its top-level color over it, and a tank's level takes the same name.
var indicatorRenames = map[string]map[string]string{
	"scale": {"color": ""},
	"tank": {
		"color":      "level_color",
		"show_caret": "caret_visible",
		"show_scale": "scale_visible",
	},
}

// invertedIndicatorFlags maps a legacy nested indicator's show flags to the hidden
// flags of the symbol that now extends the indicator.
var invertedIndicatorFlags = map[string]map[string]string{
	"scale": {
		"show_fill":  "level_hidden",
		"show_caret": "caret_hidden",
		"show_scale": "scale_hidden",
	},
	"tank": {"show_fill": "level_hidden"},
}

// invertedFlags maps each variant's legacy enabling flags to the disabling flags that
// replaced them.
var invertedFlags = map[string]map[string]string{
	"text_box":           {"auto_fit": "auto_fit_disabled"},
	"off_page_reference": {"dbl_click_nav": "dbl_click_nav_disabled"},
}

// invertedControlFlags maps a legacy control state's show flags to its hidden flags.
var invertedControlFlags = map[string]string{
	"show":           "hidden",
	"show_chip":      "chip_hidden",
	"show_indicator": "indicator_hidden",
}

// invertLegacyFlags rewrites the legacy enabling flags of a config and its control
// state as the disabling flags that replaced them, in place on the normalized wire map.
func invertLegacyFlags(cfg map[string]any) {
	variant, _ := cfg["variant"].(string)
	invertFlags(cfg, invertedFlags[variant])
	if ctl, ok := cfg["control"].(map[string]any); ok {
		invertFlags(ctl, invertedControlFlags)
	}
}

// invertFlags replaces each flag in m that flags names with its negation under the new
// name. A flag that is not a boolean is dropped.
func invertFlags(m map[string]any, flags map[string]string) {
	for from, to := range flags {
		val, ok := m[from]
		if !ok {
			continue
		}
		delete(m, from)
		if b, ok := val.(bool); ok {
			m[to] = !b
		}
	}
}

// renameColors rewrites a config's legacy color fields to the names of the parts they
// paint, and lifts a scale's or tank's nested indicator to the top of the config, in
// place on the normalized wire map. A legacy field that paints two parts is copied to
// both, so the symbol renders as it did.
func renameColors(cfg map[string]any) {
	variant, _ := cfg["variant"].(string)
	if nested, ok := legacyIndicators[variant]; ok {
		liftIndicator(
			cfg, nested, indicatorRenames[variant], invertedIndicatorFlags[variant],
		)
	}
	renames, ok := colorRenames[variant]
	if !ok {
		renames = map[string][]string{"color": {"stroke_color"}}
	}
	for from, to := range renames {
		val, ok := cfg[from]
		if !ok {
			continue
		}
		delete(cfg, from)
		for _, name := range to {
			cfg[name] = val
		}
	}
}

// liftIndicator moves the fields of the indicator nested under key to the top of cfg,
// renaming or dropping the ones renames names and negating the ones inverted names. A
// dropped field maps to "".
func liftIndicator(
	cfg map[string]any,
	key string,
	renames map[string]string,
	inverted map[string]string,
) {
	indicator, ok := cfg[key].(map[string]any)
	delete(cfg, key)
	if !ok {
		return
	}
	invertFlags(indicator, inverted)
	for k, val := range indicator {
		if name, renamed := renames[k]; renamed {
			if name == "" {
				continue
			}
			k = name
		}
		cfg[k] = val
	}
}

// withDefaults decodes a normalized config over its variant's schema defaults. A stored
// field keeps its value even when it is zero.
func withDefaults(stored msgpack.EncodedJSON) (ElementConfig, error) {
	defaults, err := defaultConfig(stored)
	if err != nil {
		return ElementConfig{}, err
	}
	fields, err := ElementConfigFields(defaults)
	if err != nil {
		return ElementConfig{}, err
	}
	mergeFields(fields, stored)
	return DecodeElementConfig(fields)
}

// defaultConfig returns the schema defaults of the variant a normalized config names,
// including those of its control state when the config carries one.
func defaultConfig(stored msgpack.EncodedJSON) (ElementConfig, error) {
	probe := msgpack.EncodedJSON{"variant": stored["variant"]}
	// The control state is the only optional field whose type declares defaults.
	if _, ok := stored["control"].(map[string]any); ok {
		probe["control"] = map[string]any{}
	}
	cfg, err := DecodeElementConfig(probe)
	if err != nil {
		return ElementConfig{}, err
	}
	cfg.ApplyDefaults()
	return cfg, nil
}

// mergeFields writes every field of src into dst, merging nested objects by field.
func mergeFields(dst, src map[string]any) {
	for k, val := range src {
		if d, ok := dst[k].(map[string]any); ok {
			if s, ok := val.(map[string]any); ok {
				mergeFields(d, s)
				continue
			}
		}
		dst[k] = val
	}
}
