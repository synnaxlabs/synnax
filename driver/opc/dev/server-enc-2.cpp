#include <open62541/client_highlevel.h>
#include <open62541/plugin/log_stdout.h>
#include <open62541/plugin/create_certificate.h>
#include <open62541/plugin/securitypolicy.h>
#include <open62541/server.h>
#include <open62541/server_config_default.h>

#include <signal.h>
#include <stdlib.h>
#include <open62541/types.h>
#include <stdio.h>
#include <errno.h>

/* sleep_ms */
#ifdef _WIN32
# include <synchapi.h>
# define sleep_ms(ms) Sleep(ms)
#else
# include <unistd.h>
# define sleep_ms(ms) usleep(ms * 1000)
#endif

/* loadFile parses the certificate file. */
static UA_INLINE UA_ByteString
loadFile(const char *const path) {
    UA_ByteString fileContents = UA_STRING_NULL;

    /* Open the file */
    FILE *fp = fopen(path, "rb");
    if(!fp) {
        errno = 0; /* We read errno also from the tcp layer... */
        return fileContents;
    }

    /* Get the file length, allocate the data and read */
    fseek(fp, 0, SEEK_END);
    fileContents.length = (size_t)ftell(fp);
    fileContents.data = (UA_Byte *)UA_malloc(fileContents.length * sizeof(UA_Byte));
    if(fileContents.data) {
        fseek(fp, 0, SEEK_SET);
        size_t read = fread(fileContents.data, sizeof(UA_Byte), fileContents.length, fp);
        if(read != fileContents.length)
            UA_ByteString_clear(&fileContents);
    } else {
        fileContents.length = 0;
    }
    fclose(fp);

    return fileContents;
}

static UA_INLINE UA_StatusCode
writeFile(const char* const path, const UA_ByteString buffer) {
    FILE *fp = NULL;

    fp = fopen(path, "wb");
    if(fp == NULL)
        return UA_STATUSCODE_BADINTERNALERROR;

    for(UA_UInt32 bufIndex = 0; bufIndex < buffer.length; bufIndex++) {
        int retVal = fputc(buffer.data[bufIndex], fp);
        if(retVal == EOF) {
            fclose(fp);
            return UA_STATUSCODE_BADINTERNALERROR;
        }
    }

    fclose(fp);
    return UA_STATUSCODE_GOOD;
}

UA_Boolean running = true;
static void stopHandler(int sig) {
    UA_LOG_INFO(UA_Log_Stdout, UA_LOGCATEGORY_USERLAND, "received ctrl-c");
    running = false;
}

int main(int argc, char* argv[]) {
    signal(SIGINT, stopHandler);
    signal(SIGTERM, stopHandler);

    UA_ByteString certificate = loadFile("server.der");
    UA_ByteString privateKey = loadFile("server_key.der");
    UA_ByteString trustList = loadFile("ca.der");

    UA_Server *server = UA_Server_new();
    UA_ServerConfig *config = UA_Server_getConfig(server);
    config->applicationDescription.applicationUri = UA_STRING("urn:open62541.server.application");

    UA_StatusCode retval = UA_ServerConfig_setDefaultWithSecurityPolicies(
        config, 4840, &certificate, &privateKey,
        &trustList, 1, NULL, 0, NULL, 0
    );

    UA_ByteString_clear(&certificate);
    UA_ByteString_clear(&privateKey);
    UA_ByteString_clear(&trustList);

    if(retval != UA_STATUSCODE_GOOD)
        goto cleanup;

    if(!running)
        goto cleanup;

    retval = UA_Server_run(server, &running);

 cleanup:
    UA_Server_delete(server);
    return retval == UA_STATUSCODE_GOOD ? EXIT_SUCCESS : EXIT_FAILURE;
}
