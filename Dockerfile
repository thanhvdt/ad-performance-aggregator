# Multi-stage build → tiny static image (no third-party deps, so nothing to fetch).
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/aggregator ./cmd/aggregator

FROM scratch
COPY --from=build /out/aggregator /aggregator
ENTRYPOINT ["/aggregator"]
# Example:
#   docker build -t ad-aggregator .
#   docker run --rm -v "$PWD":/data ad-aggregator --input /data/ad_data.csv --output /data/results
