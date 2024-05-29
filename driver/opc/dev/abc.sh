#!/bin/bash

# Variables
DAYS_VALID=365
CA_KEY="ca.key"
CA_CERT="ca.crt"
SERVER_KEY="server.key"
SERVER_CSR="server.csr"
SERVER_CERT="server.crt"
CLIENT_KEY="client.key"
CLIENT_CSR="client.csr"
CLIENT_CERT="client.crt"
CA_SERIAL="ca.srl"
CA_DER="ca.der"
SERVER_CERT_DER="server.der"
SERVER_KEY_DER="server.key.der"
CLIENT_CERT_DER="client.der"
CLIENT_KEY_DER="client.key.der"
CONFIG_FILE="openssl.cnf"

# Information for the certificates
COUNTRY="US"
STATE="California"
LOCALITY="San Francisco"
ORGANIZATION="Example Corp"
ORG_UNIT="IT Department"
EMAIL="admin@example.com"
SERVER_CN="server.example.com"
CLIENT_CN="client.example.com"
HOSTNAME="Emilianos-MacBook-Pro-2.local"
APP_URI="urn:myselfsignedserver@Emilianos-MacBook-Pro-2.local"

# Create an OpenSSL config file with SAN and ApplicationUri
cat > $CONFIG_FILE <<EOL
[req]
distinguished_name = req_distinguished_name
req_extensions = req_ext
x509_extensions = v3_ca # The extensions to add to the self-signed cert

[req_distinguished_name]
countryName = Country Name (2 letter code)
countryName_default = $COUNTRY
stateOrProvinceName = State or Province Name (full name)
stateOrProvinceName_default = $STATE
localityName = Locality Name (eg, city)
localityName_default = $LOCALITY
organizationName = Organization Name (eg, company)
organizationName_default = $ORGANIZATION
organizationalUnitName = Organizational Unit Name (eg, section)
organizationalUnitName_default = $ORG_UNIT
commonName = Common Name (eg, fully qualified host name)
commonName_max = 64

[req_ext]
subjectAltName = @alt_names

[v3_ca]
subjectAltName = @alt_names

[alt_names]
DNS.1 = $SERVER_CN
DNS.2 = $CLIENT_CN
DNS.3 = $HOSTNAME
URI.1 = $APP_URI
EOL

# Function to generate a certificate and key
generate_cert_and_key() {
    local key_file=$1
    local csr_file=$2
    local cert_file=$3
    local cn=$4

    # Generate the private key without encryption
    openssl genpkey -algorithm RSA -out $key_file

    # Create the CSR
    openssl req -new -key $key_file -out $csr_file -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$cn/emailAddress=$EMAIL" -config $CONFIG_FILE

    # Sign the CSR with the CA certificate
    openssl x509 -req -in $csr_file -CA $CA_CERT -CAkey $CA_KEY -CAcreateserial -out $cert_file -days $DAYS_VALID -sha256 -extfile $CONFIG_FILE -extensions req_ext
}

# Generate the CA private key without encryption
openssl genpkey -algorithm RSA -out $CA_KEY

# Create the CA certificate
openssl req -x509 -new -nodes -key $CA_KEY -sha256 -days $DAYS_VALID -out $CA_CERT -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=CA/emailAddress=$EMAIL" -config $CONFIG_FILE -extensions v3_ca

# Generate the server certificate and key
generate_cert_and_key $SERVER_KEY $SERVER_CSR $SERVER_CERT $SERVER_CN

# Generate the client certificate and key
generate_cert_and_key $CLIENT_KEY $CLIENT_CSR $CLIENT_CERT $CLIENT_CN

# Verify the certificates
echo "Verifying CA certificate:"
openssl x509 -in $CA_CERT -text -noout

echo "Verifying server certificate:"
openssl x509 -in $SERVER_CERT -text -noout

echo "Verifying client certificate:"
openssl x509 -in $CLIENT_CERT -text -noout

# Convert certificates and keys to DER format
echo "Converting CA certificate to DER format..."
openssl x509 -outform der -in $CA_CERT -out $CA_DER

echo "Converting server certificate to DER format..."
openssl x509 -outform der -in $SERVER_CERT -out $SERVER_CERT_DER

echo "Converting server private key to DER format..."
openssl pkcs8 -topk8 -inform PEM -outform DER -in $SERVER_KEY -out $SERVER_KEY_DER -nocrypt

echo "Converting client certificate to DER format..."
openssl x509 -outform der -in $CLIENT_CERT -out $CLIENT_CERT_DER

echo "Converting client private key to DER format..."
openssl pkcs8 -topk8 -inform PEM -outform DER -in $CLIENT_KEY -out $CLIENT_KEY_DER -nocrypt

echo "Certificate and key generation complete, including DER conversion."
