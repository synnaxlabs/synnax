#
# Copyright 2024 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.
#

bazel run //driver/opc:client opc.tcp://0.0.0.0:4840 /Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/client_cert.pem /Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/client_key.pem /Users/emilianobonilla/Desktop/synnaxlabs/synnax/driver/opc/certificates/server_cert.der
