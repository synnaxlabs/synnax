#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Directory to store generated certificates and keys
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="$SCRIPT_DIR/certificates"

# Common Name (CN) for certificates
SERVER_CN="Open62541Server@localhost"
CLIENT_CN="Open62541Client@localhost"
APPLICATION_URI="urn:open62541.server.application"

# Generate Server Certificate and Key
openssl req -new -newkey rsa:4096 -nodes -keyout $CERT_DIR/server_key.pem -out $CERT_DIR/server_csr.pem -subj "/C=DE/O=SampleOrganization/CN=$SERVER_CN"
openssl x509 -req -in $CERT_DIR/server_csr.pem -signkey $CERT_DIR/server_key.pem -out $CERT_DIR/server_cert.pem -days 365 -extfile <(printf "subjectAltName=URI:$APPLICATION_URI")
openssl x509 -outform der -in $CERT_DIR/server_cert.pem -out $CERT_DIR/server_cert.der
openssl rsa -in $CERT_DIR/server_key.pem -outform der -out $CERT_DIR/server_key.der

# Generate Client Certificate and Key
openssl req -new -newkey rsa:4096 -nodes -keyout $CERT_DIR/client_key.pem -out $CERT_DIR/client_csr.pem -subj "/C=DE/O=SampleOrganization/CN=$CLIENT_CN"
openssl x509 -req -in $CERT_DIR/client_csr.pem -signkey $CERT_DIR/client_key.pem -out $CERT_DIR/client_cert.pem -days 365 -extfile <(printf "subjectAltName=URI:$APPLICATION_URI")
openssl x509 -outform der -in $CERT_DIR/client_cert.pem -out $CERT_DIR/client_cert.der
openssl rsa -in $CERT_DIR/client_key.pem -outform der -out $CERT_DIR/client_key.der

# Cleanup CSR files
rm $CERT_DIR/server_csr.pem $CERT_DIR/client_csr.pem

# Display the generated files
echo "Generated the following files in the $CERT_DIR directory:"
echo "Server Certificate (PEM): $CERT_DIR/server_cert.pem"
echo "Server Key (PEM): $CERT_DIR/server_key.pem"
echo "Server Certificate (DER): $CERT_DIR/server_cert.der"
echo "Server Key (DER): $CERT_DIR/server_key.der"
echo "Client Certificate (PEM): $CERT_DIR/client_cert.pem"
echo "Client Key (PEM): $CERT_DIR/client_key.pem"
echo "Client Certificate (DER): $CERT_DIR/client_cert.der"
echo "Client Key (DER): $CERT_DIR/client_key.der"

# Instructions for using the generated files
echo
echo "To use the generated files, use the following command line arguments:"
echo
echo "Server:"
echo "bazel run //driver/opc/dev:server $CERT_DIR/server_cert.der $CERT_DIR/server_key.der $CERT_DIR/client_cert.der"
echo
echo "Client:"
echo "bazel run //driver/opc/dev:client opc.tcp://localhost:4840 $CERT_DIR/client_cert.der $CERT_DIR/client_key.der $CERT_DIR/server_cert.der"
