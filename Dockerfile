# SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
#
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.23 as prep

RUN apk add --no-cache ca-certificates
RUN adduser \
    --disabled-password \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 65532 \
    sparrow


FROM scratch
COPY --from=prep /etc/passwd /etc/passwd
COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY sparrow ./

USER sparrow

ENTRYPOINT ["/sparrow", "run"]