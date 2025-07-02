VERSION 0.8

ARG core=github.com/formancehq/earthly:main
IMPORT $core AS core

FROM core+base-image

sources:
    WORKDIR /src
    COPY go.* .
    COPY --dir pkg cmd .
    COPY main.go .
    SAVE ARTIFACT /src

compile:
    FROM core+builder-image
    COPY (+sources/*) /src
    WORKDIR /src
    ARG VERSION=latest
    DO --pass-args core+GO_COMPILE --VERSION=$VERSION

build-image:
    FROM core+final-image
    ENTRYPOINT ["/bin/wallets"]
    CMD ["serve"]
    COPY (+compile/main) /bin/wallets
    ARG REPOSITORY=ghcr.io
    ARG tag=latest
    DO core+SAVE_IMAGE --COMPONENT=wallets --REPOSITORY=${REPOSITORY} --TAG=$tag

deploy:
    COPY (+sources/*) /src
    LET tag=$(tar cf - /src | sha1sum | awk '{print $1}')
    WAIT
        BUILD --pass-args +build-image --tag=$tag
    END
    FROM --pass-args core+vcluster-deployer-image
    RUN kubectl patch Versions.formance.com default -p "{\"spec\":{\"wallets\": \"${tag}\"}}" --type=merge

deploy-staging:
    BUILD --pass-args core+deploy-staging

openapi:
    COPY ./openapi.yaml .
    SAVE ARTIFACT ./openapi.yaml

release:
    FROM core+builder-image
    ARG mode=local
    COPY --dir . /src
    DO core+GORELEASER --mode=$mode