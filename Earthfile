VERSION 0.6

clean:
    LOCALLY
    ARG hard=false
    RUN find . -name "*_cache" -delete
    IF [ "$hard" = "true" ]
        RUN find . -name 'node_modules' -type d -prune -exec rm -rf '{}' +
    END
