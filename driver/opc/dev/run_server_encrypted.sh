# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

realpath() (
    if [[ -d $1 ]]; then
        OLDPWD=- CDPATH= cd -P -- "$1" && pwd
    else
        OLDPWD=- CDPATH= cd -P -- "${1%/*}" && printf '%s/%s\n' "$PWD" "${1##*/}"
    fi
)
SERVER_CERT=$(realpath ./certificates/server_cert.der)
SERVER_KEY=$(realpath ./certificates/server_key.der)
bazel run //driver/opc/dev:server_encrypted $SERVER_CERT $SERVER_KEY
