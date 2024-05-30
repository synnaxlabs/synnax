realpath () (
    if [[ -d $1 ]]; then
        OLDPWD=- CDPATH= cd -P -- "$1" && pwd
    else
        OLDPWD=- CDPATH= cd -P -- "${1%/*}" && printf '%s/%s\n' "$PWD" "${1##*/}"
    fi
)
SERVER_CERT=$(realpath ./certificates/server_cert.der)
SERVER_KEY=$(realpath ./certificates/server_key.der)
CLIENT_CERT=$(realpath ./certificates/client_cert.der)
bazel run //driver/opc/dev:server_encrypted $SERVER_CERT $SERVER_KEY $CLIENT_CERT
