VERSION 0.6

link:
    LOCALLY
    RUN bash -c "cd pluto && yarn link"
    RUN bash -c "cd freighter/ts && yarn link"
    RUN bash -c "cd synnax/pkg/ui && yarn link @synnaxlabs/pluto"
    RUN bash -c "cd docs/site && yarn link @synnaxlabs/pluto"
    RUN bash -c "cd client/ts && yarn link @synnaxlabs/freighter"

clean:
    LOCALLY
    ARG hard=false
    RUN rm -rvf *_cache
    RUN for matcher in *_cache *build dist .idea .docusaurus node_modules coverage .nyc_output; do find . -name "$matcher" -exec rm -rf {} +; done
