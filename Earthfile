VERSION 0.6

clean:
    LOCALLY
    ARG hard=false
    RUN for matcher in *build dist .idea .docusaurus node_modules coverage; do find . -name "$matcher" -exec rm -rf {} +; done
