# automn

A simple Twitch chat bot.

## Requirements

[Go 1.26+](https://go.dev/dl/)

## Setup

1. Clone the repo and create a `.env` file:

```
cp .env.example .env
```

2. Start the server:

```
go run .
```

3. Open [http://localhost:7310](http://localhost:7310) and follow the setup instructions to connect your Twitch account(s).


## Stopping

`Ctrl+C`

## Port

Defaults to `:7310`. Override with `HTTP_ADDR=:8080` in `.env`.
