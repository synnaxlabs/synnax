VERSION 0.6

clean:
    LOCALLY
    ARG hard=false
    RUN rm -rvf *_cache
    RUN for matcher in *-cache *_cache *build dist .idea .docusaurus node_modules coverage target; do find . -name "$matcher" -exec rm -rf {} +; done
