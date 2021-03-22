ARG BASE_IMG=golang:1.16-buster
ARG RUN_IMG=gcr.io/distroless/base

#############
# Base stage
#############
FROM ${BASE_IMG} as base

# Add an unprivileged user
ENV USER=appuser
ENV UID=10001
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --no-create-home \ 
    --shell "/sbin/nologin" \        
    --uid "${UID}" \    
    "${USER}"

#############
# Compile stage
#############
FROM ${BASE_IMG} as compile

WORKDIR /src
COPY go.mod .
RUN go mod download
RUN go mod verify

COPY . .
RUN go build -ldflags '-w -s' -o /out/metricsbin .

#############
# Run stage
#############
FROM ${RUN_IMG}

# Use unprivileged user
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group
USER appuser:appuser

COPY --from=compile /out/metricsbin /

CMD ["/metricsbin"]