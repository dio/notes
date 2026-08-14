ARG GO_VERSION=1.21.7

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
COPY go.mod go.sum main.go index.html notes.html ./
COPY notes ./notes
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /rockybars .

FROM scratch
COPY --from=build /rockybars /rockybars
EXPOSE 8080
ENTRYPOINT ["/rockybars"]
