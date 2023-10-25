VERSION 0.7

clean:
    LOCALLY
    ARG hard=false
    RUN for matcher in *build dist .idea .docusaurus node_modules coverage *-bazel *.blaze; do find . -name "$matcher" -exec rm -rf {} +; done
