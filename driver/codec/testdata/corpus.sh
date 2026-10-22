#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Fetches DBC files from the opendbc and cantools corpora at pinned commits. Both are
# MIT licensed, and dbc/NOTICE names the source of each vendored file.
#
#   corpus.sh vendored    writes the vendored files into dbc/
#   corpus.sh all <dir>   writes every DBC file of both corpora into <dir>
#
# Each file is named <corpus>_<name>.dbc. Changing a commit means updating dbc/NOTICE.

set -euo pipefail

OPENDBC_COMMIT=c3955753ae7258412814e2cdca3ee8af979e4679
CANTOOLS_COMMIT=a417da41c6b84bbb0d2bf5060445797a38b955ac

VENDORED=(
    "cantools tests/files/dbc/fd_test.dbc"
    "cantools tests/files/dbc/floating_point.dbc"
    "cantools tests/files/dbc/floating_point_use_float.dbc"
    "cantools tests/files/dbc/issue_184_extended_mux_cascaded.dbc"
    "cantools tests/files/dbc/issue_184_extended_mux_independent_multiplexors.dbc"
    "cantools tests/files/dbc/issue_184_extended_mux_multiple_values.dbc"
    "cantools tests/files/dbc/issue_636_negative_scaling.dbc"
    "cantools tests/files/dbc/j1939.dbc"
    "cantools tests/files/dbc/motohawk.dbc"
    "cantools tests/files/dbc/motohawk_fd.dbc"
    "cantools tests/files/dbc/multiplex.dbc"
    "cantools tests/files/dbc/multiplex_2.dbc"
    "cantools tests/files/dbc/multiplex_choices.dbc"
    "cantools tests/files/dbc/padding_bit_order.dbc"
    "cantools tests/files/dbc/signed.dbc"
    "opendbc opendbc/dbc/rivian_park_assist_can.dbc"
    "opendbc opendbc/dbc/tesla_model3_vehicle.dbc"
    "opendbc opendbc/dbc/tesla_powertrain.dbc"
    "opendbc opendbc/dbc/generator/toyota/toyota_nodsu_pt.dbc"
    "opendbc opendbc/dbc/generator/volkswagen/vw_meb.dbc"
)

# Prints the GitHub repository and pinned commit of a corpus.
source_of() {
    case "$1" in
        opendbc) echo "commaai/opendbc ${OPENDBC_COMMIT}" ;;
        cantools) echo "cantools/cantools ${CANTOOLS_COMMIT}" ;;
    esac
}

vendored() {
    local out corpus path repo commit
    out="$(dirname "$0")/dbc"
    for entry in "${VENDORED[@]}"; do
        read -r corpus path <<< "$entry"
        read -r repo commit <<< "$(source_of "$corpus")"
        curl -fsSL "https://raw.githubusercontent.com/${repo}/${commit}/${path}" \
            -o "${out}/${corpus}_$(basename "$path")"
    done
}

all() {
    local out=$1 tmp repo commit name
    mkdir -p "$out"
    tmp=$(mktemp -d)
    for corpus in opendbc cantools; do
        read -r repo commit <<< "$(source_of "$corpus")"
        mkdir "${tmp}/${corpus}"
        curl -fsSL "https://github.com/${repo}/archive/${commit}.tar.gz" \
            | tar xz -C "${tmp}/${corpus}" --strip-components=1
        find "${tmp}/${corpus}" -iname '*.dbc' -print0 \
            | while IFS= read -r -d '' f; do
                name=$(basename "$f")
                cp "$f" "${out}/${corpus}_${name%.*}.dbc"
            done
    done
    rm -rf "$tmp"
}

case "${1:-}" in
    vendored) vendored ;;
    all) all "${2:?usage: corpus.sh all <dir>}" ;;
    *)
        echo "usage: corpus.sh vendored | all <dir>" >&2
        exit 1
        ;;
esac
