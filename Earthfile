VERSION 0.6

link:
    LOCALLY
    RUN bash -c "cd pluto && yarn link"
    RUN bash -c "cd synnax/pkg/ui && yarn link @synnaxlabs/pluto"
    RUN bash -c "cd docs/site && yarn link @synnaxlabs/pluto"

clean:
    LOCALLY
    ARG hard=false
    RUN rm -rvf *_cache
    RUN for matcher in *_cache *build dist .idea .docusaurus; do find . -name "$matcher" -exec rm -rf {} +; done
    IF [ "$hard" = "true" ]
        RUN find . -name 'node_modules' -type d -prune -exec rm -rf '{}' +
    END